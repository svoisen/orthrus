module ibeji

go 1.23.1

replace git.sr.ht/~kota/goldmark-gemtext => /Users/svoisen/src/goldmark-gemtext

require (
	git.sr.ht/~adnano/go-gemini v0.2.6
	git.sr.ht/~kota/goldmark-gemtext v0.3.3
	git.sr.ht/~kota/goldmark-wiki v0.1.0
	github.com/BurntSushi/toml v1.4.0
	github.com/fsnotify/fsnotify v1.8.0
	github.com/gomarkdown/markdown v0.0.0-20240930133441-72d49d9543d8
	github.com/yuin/goldmark v1.7.8
)

require (
	git.sr.ht/~kota/fuckery v0.2.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
)
