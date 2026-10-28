# Showcase pair (demo video)

`kopiku-design.png` (the "design export") and `kopiku-build.png` (the "build") are
renders of `site/kopiku.html`, an original checkout design for a fictional coffee
subscription. The build has three defects, one per category; boxes are in
`truth.json`. Regenerate with:

```sh
cd demo/tools
go run ./cmd/render -page ../site/kopiku.html -variant reference -out ../showcase/kopiku-design.png
go run ./cmd/render -page ../site/kopiku.html -variant showcase  -out ../showcase/kopiku-build.png
```
