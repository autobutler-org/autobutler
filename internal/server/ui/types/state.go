package types

import "strings"

type PageState struct {
	CurrentPageName PageName
	RootDir         string
	NavLinks        []Page
	View            string
}

func NewPageState() PageState {
	return PageState{
		CurrentPageName: PageHome,
		RootDir:         "",
		NavLinks: []Page{
			newPage(PageFiles, "/files"),
			newPage(PageCalendar, "/calendar"),
			newPage(PagePhotos, "/photos"),
			newPage(PageBooks, "/books"),
		},
	}
}

func (p PageState) WithRoute(pageName PageName) PageState {
	p.CurrentPageName = pageName
	return p
}

func (p PageState) WithRootDir(rootDir string) PageState {
	if !strings.HasPrefix(rootDir, "/") {
		rootDir = "/" + rootDir
	}
	p.RootDir = rootDir
	return p
}

func (p PageState) WithView(view string) PageState {
	if view == "" {
		view = "list"
	}

	p.View = view
	return p
}
