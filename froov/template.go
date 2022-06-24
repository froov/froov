package froov

import (
	"bytes"
	"log"
	"text/template"

	"gopkg.in/yaml.v2"
)

type Builder struct {
	page,
	lessonList,
	fileList,
	addCart,
	iconList *template.Template
}
type PageInfo struct {
	Title,
	Content string
	Loader string
	Back   bool
}

func NewBuilder(src []byte) *Builder {
	var m = map[string]string{}
	e := yaml.Unmarshal(src, &m)
	if e != nil {
		panic(e)
	}

	make := func(name string, code string) *template.Template {
		t, e := template.New("navbar").Parse(code)
		if e != nil {
			panic(e)
		}
		return t
	}

	fl := m["fileList"]
	log.Print(fl)
	tm := make("fileList", fl)

	var b bytes.Buffer
	tst := []*FrontMatter{
		{
			Id:       "",
			Version:  0,
			Title:    "ONE",
			Subtitle: "",
			Sort:     "",
			Image:    "",
			Link:     "",
			MinGrade: 0,
			MaxGrade: 0,
		},
		{
			Id:       "",
			Version:  0,
			Title:    "TWO",
			Subtitle: "",
			Sort:     "",
			Image:    "",
			Link:     "",
			MinGrade: 0,
			MaxGrade: 0,
		},
	}
	tm.Execute(&b, &tst)
	log.Printf(b.String())
	return &Builder{

		page:       make("page", m["page"]),
		iconList:   make("pinList", m["pinList"]),
		fileList:   tm,
		lessonList: make("lessonList", m["lessonList"]),
		addCart:    make("addCart", m["addCart"]),
	}
}
func (d *Builder) Page(title, content string, loader string, back bool) string {
	var b bytes.Buffer
	d.page.Execute(&b, &PageInfo{
		Title:   title,
		Content: content,
		Loader:  loader,
		Back:    back,
	})
	return b.String()
}
func (d *Builder) AddCart(f *FrontMatter) string {
	var b bytes.Buffer
	d.addCart.Execute(&b, f)
	return b.String()
}

// func (d *Builder) Slide(navbar, content string, next string) string {
// 	link := next
// 	// <div class='button'>Next</div>
// 	return d.Page("📚 froov",
// 		fmt.Sprintf(`<a class="content" href="%s">%s</a>`, link, content), "")
// }

func (d *Builder) LessonSorter(pg *CompiledDocument) string {
	var b bytes.Buffer
	d.lessonList.Execute(&b, &pg.Lesson)
	return d.Page(pg.Title,
		b.String(),
		"",
		true,
	)
}
