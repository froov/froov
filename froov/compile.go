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
	sy "sigs.k8s.io/yaml"

	cp "github.com/otiai10/copy"
)

// two tasks:
// 1. Loop (or better yet, incremental) over pricing sets to create multiple
// sites. temp

// for scoped assets, maybe we should create an asset folder
// for each folder and link it there? having it in one at least
// allows us to dedupe though without hard links.
// should this be one site under temp?

// Can we drive this from sqlite? or would a csv be more appropriate?

type WalkState struct {
	Out string
	In  string

	// asset maps the name of the asset to the link
	Asset         map[string]string
	NextAsset     int
	ServiceWorker bool

	Folder map[string]*CompiledFolder
}

// link from asset is just taking a path and returning the next integer
// we might need two paths, its source and its destination.
func (w *WalkState) linkFromAsset(p string) string {
	base, ext := baseExt(p)

	if a, ok := w.Asset[base]; ok {
		return a
	}
	w.NextAsset++
	path2 := fmt.Sprintf("%s/_%d%s", w.Out, w.NextAsset, ext)
	copyFile(p, path2)
	w.Asset[base] = "/" + path.Base(path2)
	return path2
}

func baseExt(p string) (string, string) {
	ext := path.Ext(p)
	base := path.Base(p)             // this is plus extension
	base = base[:len(base)-len(ext)] // take off extension
	return base, ext
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

func replaceExt(p string, newExt string) string {
	ext := path.Ext(p)
	return p[0:len(p)-len(ext)] + newExt
}

// path is the the source, but
func (w *WalkState) compileDocument(path string, out string) *FrontMatter {
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
		log.Printf("bad frontmatter %s", path)
		return nil
	}
	bw, _ := baseExt(path)
	r.FrontMatter.name = bw

	//r.Link = r.Id + ".html"
	// to create a link we need the path start at /temp,
	// we would have to escape each component of the path
	// but here we just assume they are ok.

	r.Link = out[len(w.Out):]

	html := string(makeMd(string(rest)))
	if r.FrontMatter.Cart > 0 {
		button := builder.AddCart(r.FrontMatter)
		html = html + button
	}

	pg := builder.Page(r.Title,
		html,
		"",
		true,
	)

	// +r.Id+
	os.WriteFile(out, []byte(pg), 0666)

	// var dfm = map[string]any{}
	// e := yaml.Unmarshal([]byte(s), &dfm)
	// if e != nil {
	// 	panic(e)
	// }
	// b, e := json.Marshal(&dfm)
	// if e != nil {
	// 	panic(e)
	// }
	// if e != nil || len(b) == 0 {
	// 	log.Printf("\nbad json %e, %s\n%v", e, s, b)
	// }
	b, e := sy.YAMLToJSON([]byte(s))
	if e != nil {
		panic(e)
	}
	// this should maybe be at the root still?
	os.WriteFile(w.Out+"/"+r.Id+".json", b, 0666)

	return r.FrontMatter
}

// we need more than the frontmatter? and we need to generate this the first
// time we access it? I guess we'll get to it eventually
// how do _assets work? maybe we should not have an _asset directory, but merge them?
// still need assets

// build folder needs to allow more than one format, a list as well
// as a grid. Or maybe the grid is just styled differently?

func (w *WalkState) buildFolder(f *CompiledFolder, header string, loader string, back bool, out string) string {
	for _, o := range f.Union {
		mergePath := w.In + "/docs/" + o
		// when we compile a folder union we have a source path
		// different from the dest path. we also have to deal with naming conflicts

		mergeFolder, e := w.compileFolder(mergePath, out, 1)
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
		a, ok := w.Asset[d.name]
		if ok {
			d.Image = a
		} else {
			log.Printf("No asset for " + d.name)
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
	// we might need fileList here instead
	// depending on Template

	if len(f.Document) > 0 {
		if f.IndexJson.Template == "list" {
			log.Printf("list %d\n", len(f.Document))
			builder.fileList.Execute(&b, &f.Document)
		} else {
			builder.iconList.Execute(&b, &f.Document)
		}
	}
	return builder.Page(f.Title,
		"<div class='header'>"+header+"</div>"+b.String(),
		loader, back,
	)
}

// with an asset we only care about's source; the dest will be generated.
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

func (w *WalkState) compileFolder(in, out string, depth int) (*CompiledFolder, error) {
	os.Mkdir(out, os.ModePerm)
	f := &CompiledFolder{}
	f.Name = stem(in)
	b, e := os.ReadFile(in + "/index.json")
	json.Unmarshal(b, &f.IndexJson)
	baseOut := out[len(w.Out):]
	f.Link = baseOut + "/index.html"

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
				fc, e := w.compileFolder(in+"/"+o.Name(), out+"/"+o.Name(), depth+1)
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
			if ext == ".md" && o.Name()[0] != '_' {
				// we need to convert this to a blob
				doc := w.compileDocument(in+"/"+o.Name(), out+"/"+replaceExt(o.Name(), ".html"))
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
	welcome, _ = os.ReadFile(in + "/_index.md")
	ws := ""
	if len(welcome) > 0 {
		ws = makeMd(string(welcome))
		log.Printf("converted\n%s", string(ws))
	}

	if depth == 0 {
		loader := `<script>
			if ('serviceWorker' in navigator) {
			window.addEventListener('load', () => {
				navigator.serviceWorker.register('./sw.js');
			});
			}
			</script>`

		content := w.buildFolder(f, ws, loader, false, out)
		os.WriteFile(fmt.Sprintf("%s/index.html", out), []byte(content), 0666)
	} else {
		//crumbs := "<div class='crumb'>" + f.Title + "</div>"
		content := w.buildFolder(f, ws, "", true, out)
		//  f.IndexJson.Id
		os.WriteFile(fmt.Sprintf("%s/index.html", out), []byte(content), 0666)
		//w.linkFromHtml(content)
	}

	return f, nil
}

func frontMatter(p string) *FrontMatter {
	input, e := os.ReadFile(p)
	if e != nil {
		panic(e)
	}
	inputs := string(input)
	//inputs, e : = os.ReadFile(p)
	//rest, err := frontmatter.Parse(strings.NewReader(inputs), &r.FrontMatter)
	o1 := strings.Index(inputs, "---")
	o2 := o1 + 4 + strings.Index(inputs[o1+4:], "---")
	if o1 == -1 {
		return nil
	}
	s := inputs[o1+4 : o2]

	var r FrontMatter

	err := yaml.Unmarshal([]byte(s), &r)
	if err != nil {
		log.Printf("bad frontmatter %s", p)
		panic(err)
	}
	return &r
}

func FriendlyName(p string) string {
	dir := path.Dir(p)
	base, ext := baseExt(p)
	if ext == ".md" {
		fm := frontMatter(p)
		if fm != nil && len(fm.Subtitle) > 0 {
			base = strings.TrimSpace(fm.Subtitle)
		}
	} else if ext == ".jpeg" {
		fm := frontMatter(dir + "/../" + base + ".md")
		if fm != nil && len(fm.Subtitle) > 0 {
			base = fm.Subtitle
		}
	}

	// we want pull out only the words, putting in a hyphen in place of anything
	// not a letter
	var b bytes.Buffer
	skip := false
	for _, c := range strings.TrimSpace(strings.ToLower(base)) {
		if c == '_' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			if skip {
				b.WriteRune('-')
			}
			skip = false
			b.WriteRune(c)
		} else if !skip {
			skip = true
		}
	}
	return dir + "/" + b.String() + ext
}

func RenameFolder(in string, out string) {
	r, e := os.ReadDir(in)
	if e != nil {
		log.Printf("error %s", in)
		return
	}
	for _, o := range r {
		inf := in + "/" + o.Name()
		outf := out + "/" + o.Name()
		f := FriendlyName(inf)
		if !o.IsDir() && inf != f {
			f2 := out + "/" + path.Base(f)
			fmt.Printf("mv %s %s\n", outf, f2)
			// only rename the out copy

			os.Rename(outf, f2)
		}
		if o.IsDir() {
			RenameFolder(in+"/"+o.Name(), out+"/"+o.Name())
		}
	}
}
func Compile(in string, serviceWorker bool) {
	z, e := os.ReadFile(in + "/css.yaml")
	if e != nil {
		log.Fatalf("no css.yaml")
	}
	builder = NewBuilder(z)

	out := in + "/temp"
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

	_, e = o.compileFolder(in+"/docs", out, 0)
	if e != nil {
		panic(e)
	}
}
