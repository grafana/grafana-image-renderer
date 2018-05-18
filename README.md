# Grafana Image Renderer

A Grafana Backend Plugin that handles rendering panels &amp; dashboards to PNGs using headless chrome.

# Dependencies

Nodejs v8+ installed. 

# Installation 

- git clone into Grafana external plugins folder. 
- yarn install --pure-lockfile
- npm run build
- restart grafana-server , it should log output that the renderer plugin was found and started. 
- To get more logging info update grafana.ini section [log] , key filters = rendering:debug
