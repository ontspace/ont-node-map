Ontology Node Map

# Requirements:

* Golang >= 1.12 (need Go mod)
* Node(Npm)

# Run in shell:

* Clone repo `git clone [dir]`
* Go to frontend directory `cd [dir]/fe`, and install node requirements `npm i`
* Build frontend project `npm run build`
* Go back to top directory `cd ..`
* Execute go main file `go run main.go`

# Deploy

* Build go executable file for your platform, `go build main.go`
* Build frontend file (above)
* Copy go file and frontend `fe/dist` directory to somewhere, structure should be like this:
  ```
    ├── fe
    │   └── dist
    │       ├── css
    │       │   └── app.eeb1cbbf.css
    │       ├── favicon.ico
    │       ├── index.html
    │       └── js
    │           ├── app.77176990.js
    │           ├── app.77176990.js.map
    │           ├── canvg.0dd7511e.js
    │           ├── canvg.0dd7511e.js.map
    │           ├── chunk-vendors.6ab19f40.js
    │           ├── chunk-vendors.6ab19f40.js.map
    ├── main
    
  ```
* Run above go file

