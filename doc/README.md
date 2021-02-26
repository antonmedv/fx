# some options for document processing

ronn -r DOCS.md --pipe >doc/fx.1.man
cat fx.1.man | groff -T utf8 -man | less
glow DOCS.md > doc/page.txt
