#!/usr/bin/env sh

curl -s -X GET "https://musicbrainz.org/ws/2/genre/all?fmt=txt" \
  -H "User-Agent: music-release-publisher/1.0.0-beta ( fabasoad@gmail.com )"
