# backmirror: Fully-featured prosemirror backend

Example for Uploading files


```Bash
curl \
  -F "filename=CHANGELOG.md" \
  -F "filecomment=This is an example" \
  -F "file=@/Users/mb/Documents/UTILS/CHANGELOG.md" \
  http://192.168.1.43:5555/files/MARKDOWN
```  