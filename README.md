# Authenthication examples
+ login
`curl -X POST -F 'email=a@a.com' -F 'password=password' http://192.168.1.47:5555/login`
+ register
`curl -X POST -F 'email=a@a.com' -F 'password=password' http://192.168.1.47:5555/register`

# blext: fully-featured collaborative editing back-end

+ Persitance layer
+ File conversions with [pandoc](https://pandoc.org/)
+ WebSockets

Example for converting .md files to .pdf

```Bash
curl \
  -F "filename=CHANGELOG.md" \
  -F "email=a@a.com" \
  -F "filecomment=This is an example" \
  -F "file=~/Documents/.../CHANGELOG.md" \
  http://192.168.1.43:5555/convert/MARKDOWN
```  

