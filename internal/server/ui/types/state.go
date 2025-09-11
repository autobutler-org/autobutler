package types

type PageState struct {
	CurrentPageName PageName
	RootDir         string
	NavLinks        []Page
}

func NewPageState() PageState {
	return PageState{
		CurrentPageName: PageHome,
		RootDir:         "",
		NavLinks: []Page{
			newPage(PageChat, "/chat"),
			newPage(PageFiles, "/files"),
			newPage(PageCalendar, "/calendar"),
		},
	}
}

func (p PageState) WithRoute(pageName PageName) PageState {
	p.CurrentPageName = pageName
	return p
}

func (p PageState) WithRootDir(rootDir string) PageState {
	p.RootDir = rootDir
	return p
}
