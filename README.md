# RocketRankBot
A Twitch bot allowing viewers to fetch the broadcasters Rocket League Rank 
through chat commands in a customizable format.

## Building twirp files
Make sure `protoc` and `protoc-gen-twirp` are in your path!  
#### Commander:
```
protoc --twirp_out=./ --go_out=./ rpc/trackerggscraper/trackerggscraper.proto
```
#### TrackerGGScraper:
```
npx twirpscript
```