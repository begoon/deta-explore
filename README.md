# Deta Space runtime environment and file system exploration tool

This simple app is a tool to explore the environment and file system of Space micros.

If you want to better understand how Deta runtime works, this app definitely helps.

## Functionality

The app allows to:

1. browse the filesystem of the micro runtime.
2. see the environment variables of the micro runtime.
3. execute arbitary programs or shell scripts interactively.

### File system browser

Click on `Files` in the top menu.

The browser looks like this:

![image](https://user-images.githubusercontent.com/84461/230232378-660e5dee-9cce-4fb1-b89c-8deacacea80b.png)

For example, you can navigate to the `/var/task` directory and check, what files you actually deployed within the app.

The browser can download directories as a `tar.gz` file by clicking on ⇩.

Also, the browser can show binary files as the hex dump by clicking on 👁️. For example:

![image](https://user-images.githubusercontent.com/84461/230236214-c46f66db-1c1c-401d-be9c-2f2fb824abaa.png)

### Environment variables

Click on `Environment` in the top menu.

![image](https://user-images.githubusercontent.com/84461/230233726-f8ef6974-ee8e-4997-879d-ae38b12effef.png)

The response is JSON, and its "nicification" in the the web browser extension (`JSONVue` in this case).

### Execute arbitary programs or shell scripts

Click on `Run` in the top menu, then type one or command in the input field and press "Run".

For example:

![image](https://user-images.githubusercontent.com/84461/230234410-753c9f5f-9380-449a-acaa-85142cbbfdde.png)

## How to deploy the app

The app is written in Go, so you need to install the Go compiler. The app was
created using Go 1.20.

0. [Install Go](https://go.dev/doc/install)
1. Checkout the source code:
```
git clone git@github.com:begoon/deta-explore.git
cd deta-explore
``` 
2. Build
```
make build
```
3. Run app locally, just in case:
```
./exe 
```

or

``` 
make run
```
4. Create your Data Space deployment:
```
(cd micro && space new)
```
then follow the instructions from `space` CLI.

If `space` CLI succeeds, the `micro/.space` folder will be created.

5. Deployment
```
make deploy
```    
This comment builds the executable for specifically `linux/amd4` architecture and
then pushes it to Deta Space. Alternately, we can change the `Spacefile` to push
the sources to Deta instead, and then build the executable in Deta. The current 
`Spacefile` pushes the already built executable. 

After the push, CLI will give you the URL of your app, which you can just click on.

NOTE: In the `Spacefile`, ALL public route as disabled via `public_routes` to be `/none` ("none"
is just an arbitary nonexistent path). Be careful making your deployment public. The environment
variables viewer may expose your deta project key and the api key.

Bottom line: use the app responsibly :-)
