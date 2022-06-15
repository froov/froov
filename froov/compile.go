package froov

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v2"

	cp "github.com/otiai10/copy"
)

// for scoped assets, maybe we should create an asset folder
// for each folder and link it there? having it in one at least
// allows us to dedupe though without hard links.

type WalkState struct {
	Out string
	In  string

	// asset maps the name of the asset to the link
	Asset         map[string]string
	NextAsset     int
	ServiceWorker bool

	Folder map[string]*CompiledFolder
}

func (w *WalkState) linkFromAsset(p string) string {
	base := path.Base(p)
	ext := path.Ext(p)
	base = base[:len(base)-len(ext)]

	if a, ok := w.Asset[base]; ok {
		return a
	}
	w.NextAsset++
	path2 := fmt.Sprintf("%s/_%d%s", w.Out, w.NextAsset, ext)
	copyFile(p, path2)
	w.Asset[base] = path.Base(path2)
	return path2
}

var builder *Builder

// I probably want to hash the textbook and then create the pages as
// hash.x.x? not necessarily a win because I'm not sharing with textbooks almost
// the same. but it does let be potentially extract dynamically.
func makeMd(mdx string) string {
	md := markdown.NormalizeNewlines([]byte(mdx))
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)
	unsafe := "<div class='content'>" + string(markdown.ToHTML(md, parser, nil)) + "</div>"
	return unsafe
	// what would this break?
	//return string(bluemonday.UGCPolicy().SanitizeBytes(unsafe))
}

func (w *WalkState) compileDocument(path string) *FrontMatter {
	input, _ := os.ReadFile(path)
	inputs := string(input)
	r := &CompiledDocument{
		FrontMatter: &FrontMatter{},
		Contents:    Lesson{},
		Lesson:      []Lesson{},
	}
	//rest, err := frontmatter.Parse(strings.NewReader(inputs), &r.FrontMatter)
	o1 := strings.Index(inputs, "---")
	o2 := o1 + 4 + strings.Index(inputs[o1+4:], "---")
	if o1 == -1 {
		return nil
	}
	s := inputs[o1+4 : o2]
	rest := inputs[o2+4:]

	err := yaml.Unmarshal([]byte(s), r.FrontMatter)
	if err != nil {
		log.Print("bad frontmatter")
		return nil
	}
	r.Link = r.Id + ".html"

	html := string(makeMd(string(rest)))
	pg := builder.Page(r.Title,
		html,
		"",
		true,
	)
	os.WriteFile(w.Out+"/"+r.Id+".html", []byte(pg), 0666)
	return r.FrontMatter
}

// we need more than the frontmatter? and we need to generate this the first
// time we access it? I guess we'll get to it eventually
// how do _assets work? maybe we should not have an _asset directory, but merge them?
// still need assets

func (w *WalkState) buildFolder(f *CompiledFolder, header string, loader string, back bool) string {
	for _, o := range f.Union {
		mergePath := w.In + "/docs/" + o
		mergeFolder, e := w.compileFolder(mergePath, 1)
		if e != nil {
			log.Printf("error merging %s", mergePath)
		}
		f.Document = append(f.Document, mergeFolder.Document...)
	}
	// we need to substitute images from _assets; folder may want
	// different icons than default
	for _, d := range f.Pin {
		a, ok := w.Asset[d.Title]
		if ok {
			d.Image = a
		} else {
			log.Printf("No asset for " + d.Title)
		}
	}
	for _, d := range f.Document {
		a, ok := w.Asset[d.Title]
		if ok {
			d.Image = a
		} else {
			log.Printf("No asset for " + d.Title)
		}
	}
	for _, d := range f.Folder {
		a, ok := w.Asset[d.Name]
		if ok {
			d.Image = a
		} else if !d.Hidden {
			log.Printf("No asset for " + d.Name)
		}
	}
	var b bytes.Buffer

	log.Printf("Building %s", f.Title)
	if len(f.Pin) > 0 {

		// this puts pins in if we have them.
		// pins can be either folders or documents, and we don't
		// really have a way to distinguish them.
		// we have to map the pins to links, this should be
		// json things.
		sort.Slice(f.Pin, func(i, j int) bool {
			return f.Pin[i].Sort < f.Pin[j].Sort
		})
		builder.iconList.Execute(&b, &f.Pin)
	}
	sort.Slice(f.Folder, func(i, j int) bool {
		return f.Folder[i].Sort < f.Folder[j].Sort
	})
	// this is the body of the page
	sort.Slice(f.Document, func(i, j int) bool {
		return f.Document[i].Sort < f.Document[j].Sort
	})
	if len(f.Folder) > 0 {
		builder.iconList.Execute(&b, &f.Folder)
	}
	if len(f.Document) > 0 {
		builder.iconList.Execute(&b, &f.Document)
	}
	return builder.Page(f.Title,
		"<div class='header'>"+header+"</div>"+b.String(),
		loader, back,
	)
}

func (w *WalkState) compileAssets(path string) {
	o, e := os.ReadDir(path)
	if e != nil {
		log.Printf("error reading %s,%v", path, e)
		return
	}
	for _, j := range o {
		if !j.IsDir() {
			w.linkFromAsset(path + "/" + j.Name())
		}
	}
}

func (w *WalkState) compileFolder(in string, depth int) (*CompiledFolder, error) {
	f := &CompiledFolder{}
	f.Name = stem(in)
	b, e := os.ReadFile(in + "/index.json")
	json.Unmarshal(b, &f.IndexJson)
	f.Link = f.Id + ".html"

	if f, ok := w.Folder[in]; ok {
		return f, nil
	}

	w.Folder[in] = f
	de, e := os.ReadDir(in)
	if e != nil {
		return nil, e
	}

	for _, o := range de {
		if o.IsDir() {
			// if there is no index.json then this is an asset folder
			// make everything into assets that are scoped to this folder
			if exists(in + "/" + o.Name() + "/index.json") {
				// this is a datum, we need to compile it.
				// we need to use its name to attach an asset it to it.
				fc, e := w.compileFolder(in+"/"+o.Name(), depth+1)
				if e != nil {
					return nil, e
				}
				if !fc.Hidden {
					f.Folder = append(f.Folder, fc)
				}
			} else if o.Name() == "_asset" {
				w.compileAssets(in + "/" + o.Name())
			}

		} else {
			ext := path.Ext(o.Name())
			if ext == ".md" {
				// we need to convert this to a blob
				doc := w.compileDocument(in + "/" + o.Name())
				if doc == nil {
					//log.Printf("error %s", in+"/"+o.Name())
				} else {
					f.Document = append(f.Document, doc)
				}
			}
		}
	}

	if f.Hidden {
		return f, nil
	}

	welcome := []byte{}
	welcome, _ = os.ReadFile(in + "/index.md")

	if depth == 0 {
		loader := `<script>
			if ('serviceWorker' in navigator) {
			window.addEventListener('load', () => {
				navigator.serviceWorker.register('/sw.js');
			});
			}
			</script>`
		ws := ""
		if len(welcome) > 0 {
			ws = makeMd(string(welcome))
		}
		content := w.buildFolder(f, ws, loader, false)
		os.WriteFile(fmt.Sprintf("%s/index.html", w.Out), []byte(content), 0666)
	} else {
		//crumbs := "<div class='crumb'>" + f.Title + "</div>"
		content := w.buildFolder(f, "", "", true)
		os.WriteFile(fmt.Sprintf("%s/%s.html", w.Out, f.IndexJson.Id), []byte(content), 0666)
		//w.linkFromHtml(content)
	}

	return f, nil
}

func Compile(in string, serviceWorker bool) {
	z, e := os.ReadFile(in + "/css.yaml")
	if e != nil {
		log.Fatalf("no css.yaml")
	}
	builder = NewBuilder(z)

	out := in + "/froov"
	os.RemoveAll(out)
	os.Mkdir(out, os.ModePerm)
	cp.Copy(in+"/public", out)

	o := WalkState{
		Out:           out,
		In:            in,
		Asset:         map[string]string{},
		NextAsset:     0,
		ServiceWorker: serviceWorker,
		Folder:        map[string]*CompiledFolder{},
	}

	_, e = o.compileFolder(in+"/docs", 0)
	if e != nil {
		panic(e)
	}
}
