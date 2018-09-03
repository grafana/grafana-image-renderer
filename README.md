# Grafana Image Renderer

A Grafana Backend Plugin that handles rendering panels &amp; dashboards to PNGs using headless chrome.

# Dependencies

Nodejs v8+ installed. 

# Installation 

- git clone into Grafana external plugins folder. 
- yarn install --pure-lockfile
- yarn run build
- restart grafana-server , it should log output that the renderer plugin was found and started. 
- To get more logging info update grafana.ini section [log] , key filters = rendering:debug


# Remote Rendering Docker image

A dockerfile is provided for deploying the remote-image-renderer in a container.
You can then configure your Grafana server to use the container via the 
```
[rendering]
server_url=http://renderer:8081/render
```
config setting in grafana.ini

A docker-compose example is provided in docker/
to launch

```
cd docker
docker-compose up
```


