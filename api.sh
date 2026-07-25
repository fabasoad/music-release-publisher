#!/usr/bin/env sh

user_agent="music-release-publisher/1.0.0-beta ( fabasoad@gmail.com )"

get_genres() {
  curl -s -X GET "https://musicbrainz.org/ws/2/genre/all?fmt=txt" \
    -H "User-Agent: ${user_agent}"
}

get_releases() {
  genres_query=$(awk 'NR==1{printf "tag:\"%s\"", $0} NR>1{printf " OR tag:\"%s\"", $0}' genres.txt | python3 -c "import sys, urllib.parse; print(urllib.parse.quote(sys.stdin.read()))")
  url="https://musicbrainz.org/ws/2/release?query=date:2026-07-24%20AND%20(${genres_query})&fmt=json"
  curl -s -X GET "${url}" -H "User-Agent: ${user_agent}"
}

get_releases
