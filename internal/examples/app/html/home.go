package html

import (
	. "alin.ovh/gomponents"
	. "alin.ovh/gomponents/html"
)

func HomePage(items []string) Node {
	return page(
		"Home",
		H1(Text("Home")),

		P(Text("This is a gomponents example app!")),

		P(
			Raw(
				`Have a look at the <a href="https:/github.com/alanpearce/gomponents/tree/main/internal/examples/app">source code</a> to see how it’s structured.`,
			),
		),

		Ul(Map(items, func(s string) Node {
			return Li(Text(s))
		})),
	)
}
