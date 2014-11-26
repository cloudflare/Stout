set -e

# Install jq with brew install jq

RELEASE=$1

git tag $RELEASE
git push origin master --tags

UPLOAD_URL=$(curl -X POST "https://api.github.com/repos/EagerIO/Stout/releases" \
  -H "Accept: application/vnd.github.v3+json" \
  -H "Authorization: token $GITHUB_AUTH" \
  -H "Content-Type: application/json" \
  -d "
{
  \"tag_name\": \"$RELEASE\"
}" | jq -r '.upload_url' | cut -d { -f 1)

mkdir -p debian

echo "
Package: stout
Source: stout
Version: $RELEASE
Architecture: all
Maintainer: Zack Bloom <zack@eager.io>
Description: The reliable static website deploy tool
" > `dirname $0`/../control

`dirname $0`/xc.sh

upload () {
  local archive=$1
  local filename=$(basename "$archive")
  local extension="${filename##*.}"

  if [ "$extension" == "md" ]; then
    return
  fi

  curl -X POST "$UPLOAD_URL?name=$filename" \
    -H "Content-Type: application/octet-stream" \
    -H "Authorization: token $GITHUB_AUTH" \
    --data-binary @$archive
}

for f in builds/snapshot/*; do upload "$f" & done

wait
