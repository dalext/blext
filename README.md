# backmirror: Fully-featured prosemirror backend

Example for Uploading files


```Bash
curl \
  -F "filename=FileTitle.exe" \
  -F "filecomment=This is an example" \
  -F "file=@/Users/mb/Documents/UTILS/signP2DMapk.sh" \
  http://192.168.1.43:5555/files/WordType
```  