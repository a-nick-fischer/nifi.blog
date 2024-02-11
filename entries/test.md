---
title: Writing your own SSR for Dummies
summary: We didn't have enough SSR tools yet, so I decided to write my own and learn Go while doing it.
date: 11.02.2024
tags:
    - ssr
    - golang
    - webdev
---

# Writing your own SSR for Dummies
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer et tempor elit. Etiam et ligula non elit sollicitudin egestas ut pellentesque felis. Sed ligula arcu, auctor quis lacus a, pharetra lobortis enim. Cras pharetra rhoncus arcu. Phasellus vitae eros ac quam vestibulum facilisis sed sit amet nisl. Phasellus in ante tellus. Aliquam nulla mauris, fermentum quis risus at, aliquet sodales tellus. Suspendisse cursus elit ut risus condimentum consectetur. Integer tortor urna, feugiat a tincidunt id, accumsan in ante. Fusce sollicitudin tempus risus, sed mollis nisl tempus at. Sed consequat nulla lectus, vitae tempor enim porta maximus. Cras pellentesque lacinia ligula, eu rhoncus ex iaculis non. Nullam ultrices lorem quis augue dictum scelerisque at vel metus. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Morbi auctor justo eget felis fringilla convallis. Aenean ligula dui, suscipit et congue faucibus, vestibulum ut lectus.

## Meet Jeff
Phasellus lacinia turpis sem, in pulvinar massa volutpat nec. Nulla quis sagittis lorem. Aenean odio purus, accumsan id nibh a, elementum rhoncus dui. Maecenas ornare, nisi ac sollicitudin dapibus, purus diam faucibus magna, pulvinar suscipit diam orci interdum enim. Morbi nec odio id augue accumsan luctus. Fusce risus augue, tempor quis finibus a, viverra nec leo. Maecenas euismod, tellus sit amet laoreet consequat, orci libero interdum augue, id blandit quam ex eleifend urna. Etiam diam quam, vehicula eget suscipit in, volutpat in quam. Nunc hendrerit purus erat, eu condimentum diam imperdiet bibendum. Integer tempus ipsum metus, ac pharetra eros mollis porta.

```go
func main() {
	articles := readArticles()
	photos := readPhotos()
	templates := readTemplates()

	regenerateOutputDir()

	generateThumbnails(photos)
	generateBlogEntries(articles, templates)
	generateTemplate(templates, BLOG_TEMPLATE_FILE, articles)
	generateTemplate(templates, PHOTOS_TEMPLATE_FILE, photos)
	generateTemplate(templates, INDEX_TEMPLATE_FILE, "")
	copyFiles()
}
```

Suspendisse id finibus libero. Vivamus sagittis quam eget dui egestas lobortis. Phasellus pulvinar blandit elit, eu rhoncus lacus luctus pulvinar. Vivamus consequat elit mollis ex pretium pretium. Suspendisse sit amet urna mauris. Phasellus rutrum justo congue tellus feugiat malesuada. Nulla facilisi. Ut in urna blandit, finibus ante ut, sodales nisl. Duis blandit ligula pretium odio aliquam accumsan. Sed eu blandit felis. Sed laoreet iaculis massa, ut elementum libero varius ut. Praesent auctor purus a arcu vestibulum pulvinar. Pellentesque eget tellus scelerisque, faucibus ipsum eget, posuere neque. Nunc ultricies consectetur felis, quis consequat elit auctor fringilla. Aliquam suscipit non ex in posuere. Duis id ullamcorper sem.

Sed dui eros, ultrices vel purus ac, consectetur euismod enim. Donec accumsan est a posuere viverra. Proin aliquet faucibus eros, ac blandit eros ullamcorper nec. Ut vitae volutpat arcu. Quisque lectus mauris, pellentesque vel sodales non, viverra eu enim. Sed vitae cursus mi. Curabitur mollis nulla ipsum, et rutrum libero feugiat vitae. Nulla bibendum lacinia lacus nec bibendum. Quisque ultrices velit dolor, ac porttitor justo bibendum ut. Vivamus finibus ante nibh, sed eleifend nisl tincidunt eu. Phasellus nec sagittis elit, non facilisis sapien. Mauris sagittis fermentum massa, in tristique lectus convallis sit amet. Maecenas vehicula eros dignissim lobortis porttitor. Morbi viverra est ac enim sollicitudin elementum. Curabitur tortor tortor, auctor at dapibus non, bibendum eu dui. 