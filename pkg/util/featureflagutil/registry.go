package featureflagutil

// registry declares every beta flag. Adding a beta is one entry here plus its
// gate checks. Retiring one is deleting its entry and gates, and appending a
// settingsutil migration that drops its key.
var registry = []Flag{}
