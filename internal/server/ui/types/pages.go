package types

type PageName string

const (
	PageBooks    PageName = "Books"
	PageCalendar PageName = "Calendar"
	PageDevices  PageName = "Devices"
	PageFiles    PageName = "Files"
	PageHome     PageName = "Home"
	PagePhotos   PageName = "Photos"
	PageHealth   PageName = "Health"
)

type Page struct {
	Name PageName
	Href string
}

func newPage(name PageName, href string) Page {
	return Page{
		Name: name,
		Href: href,
	}
}
