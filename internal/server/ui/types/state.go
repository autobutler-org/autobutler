package types

type PageState struct {
	CurrentRoute RouteName
	RootDir      string
}

func NewPageState() PageState {
	return PageState{
		CurrentRoute: RouteHome,
		RootDir:      "",
	}
}

func (p PageState) WithRoute(route RouteName) PageState {
	p.CurrentRoute = route
	return p
}

func (p PageState) WithRootDir(rootDir string) PageState {
	p.RootDir = rootDir
	return p
}
