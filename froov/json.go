package froov

type CompiledFolder struct {
	Name string
	IndexJson
	Document []*FrontMatter
	Folder   []*CompiledFolder
}
type CompiledDocument struct {
	*FrontMatter

	// these can be discarded after building the document to save memory.
	Contents Lesson
	Lesson   []Lesson
}

// the front matter from a folder comes from index.json
type IndexJson struct {
	Id       string      `json:"id,omitempty" yaml:"id"`
	Version  int         `json:"version,omitempty" yaml:"version"`
	Title    string      `json:"title,omitempty" yaml:"title"` // this is the name of the folder and its sort key
	Sort     string      `json:"sort,omitempty"`
	Image    string      `json:"image,omitempty" yaml:"sort"`
	Link     string      `json:"link,omitempty"`  // not loaded, computed after a load
	Union    []string    `json:"union,omitempty"` // union all these (probably hidden) folders
	Hidden   bool        `json:"hidden,omitempty"`
	Pin      []*LinkInfo `json:"pin,omitempty"`
	Template string      `json:"template,omitempty"`
}
type FrontMatter struct {
	Id       string   `yaml:"id"`
	Version  int      `yaml:"version"`
	Title    string   `yaml:"title"` // this is the name of the folder and its sort key
	Subtitle string   `yaml:"subtitle"`
	Sort     string   `yaml:"sort"`
	Image    string   `yaml:"image"`
	Link     string   `yaml:"link"` // not loaded, computed after a load
	MinGrade int      `yaml:"minGrade"`
	MaxGrade int      `yaml:"maxGrade"`
	IronShop []string `yaml:"ironShop"`
	Cart     int      `yaml:"cart"`
	name     string
}

type LinkInfo struct {
	Title   string `json:"title,omitempty"`
	Sort    string `json:"sort,omitempty"`
	Image   string `json:"image,omitempty"` // just ignored? we are using the image from the linked from, this could be default though.
	Path    string `json:"path,omitempty"`
	Folder  bool   `json:"folder,omitempty"`
	Content string `json:"content,omitempty"`
	//Link  string
}

// if we need it the lesson title is on the first slide #
type Lesson struct {
	Number int
	Slide  []string
	Link   string
}

// sorting the grade strings is going to go wrong :(
// should we sort before creating the json then?
// folders are like subject links too.
// can we  capture that subject link here to use it as a link (instead of the
// subject hash)
// type Folder struct {
// 	Title   string
// 	Welcome string
// 	// how do I turn a subject link into a href?
// 	Pin    []*SubjectLinkJson
// 	More   []*SubjectLinkJson
// 	Folder map[string]*Folder
// 	Link   *SubjectLinkJson
// }

/*
type SchoolTheme struct {
	Image map[string]string
}


type SchoolJson struct {
	Welcome string
	Subject []*SubjectLinkJson
}

type SubjectLinkJson struct {
	Title string
	Sort  string
	//Hash  string
	Image   string
	Path    string
	Pin     bool
	Folder  bool
	Content string
	// we don't load this, we set it when build the page.
	Link string
}
*/

// type SubjectLinkJson struct {
// 	Title string
// 	Sort  string
// 	Hash  string
// 	Image string
// 	Path  string
// 	Pin   bool
// 	Link  string
// }
