# Sitemapper

Builds a Graphviz graph description of the sitemap for a given site.

## Usage

```bash
sitemapper https://some.url ./sitemap.dot
dot -Tpdf ./sitemap.dot -o sitemap.pdf
```
