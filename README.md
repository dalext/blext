# backmirror: Fully-featured prosemirror backend

Example for converting .md files to .pdf

```Bash
curl \
  -F "filename=CHANGELOG.md" \
  -F "email=a@a.com" \
  -F "filecomment=This is an example" \
  -F "file=@/Users/mb/Documents/UTILS/CHANGELOG.md" \
  http://192.168.1.43:5555/convert/MARKDOWN
```  
