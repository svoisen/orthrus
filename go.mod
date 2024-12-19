module orthrus

go 1.23.1

replace git.sr.ht/~kota/goldmark-gemtext => /Users/svoisen/src/goldmark-gemtext

require (
	git.sr.ht/~adnano/go-gemini v0.2.6
	git.sr.ht/~kota/goldmark-gemtext v0.3.3
	git.sr.ht/~kota/goldmark-wiki v0.1.0
	github.com/BurntSushi/toml v1.4.0
	github.com/fsnotify/fsnotify v1.8.0
	github.com/yuin/goldmark v1.7.8
	github.com/yuin/goldmark-highlighting v0.0.0-20220208100518-594be1970594
	go.abhg.dev/goldmark/wikilink v0.5.0
)

require (
	git.sr.ht/~kota/fuckery v0.2.0 // indirect
	github.com/alecthomas/chroma v0.10.0 // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	go.abhg.dev/goldmark/frontmatter v0.2.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
