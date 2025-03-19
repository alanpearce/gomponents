package import_test

import (
	"testing"

	. "go.alanpearce.eu/gomponents"
	. "go.alanpearce.eu/gomponents/components"
	. "go.alanpearce.eu/gomponents/html"
	. "go.alanpearce.eu/gomponents/http"
)

func TestImports(t *testing.T) {
	t.Run(
		"this is just a test that does nothing, but I need the dot imports above",
		func(t *testing.T) {
			_ = El("div")
			_ = A()
			_ = HTML5(HTML5Props{})
			_ = Adapt(nil)
		},
	)
}
