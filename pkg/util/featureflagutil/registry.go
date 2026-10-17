package featureflagutil

// registry declares every beta flag. Adding a beta is one entry here plus its
// gate checks. Retiring one is deleting its entry and gates, and appending a
// settingsutil migration that drops its key.
var registry = []Flag{
	{
		Key:   Slides,
		Label: "Slides",
		Description: "The presentation editor, with PowerPoint import and export. Turning it off hides " +
			"Slides from everyone; stored presentations are kept, not deleted. It is not a security measure.",
		Default:      true,
		IntroducedIn: "#1152",
		SunsetIssue:  2869,
	},
}
