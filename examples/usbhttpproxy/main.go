// https://blog.golang.org/examples
package main

import (
	"log"
	"net/http"

	go_usbmuxd_device "github.com/electricbubble/go-usbmuxd-device"
)

func main() {
	hproxy, err := go_usbmuxd_device.NewUSBHTTPProxy(nil)
	if err != nil {
		log.Fatal(err)
	}
	http.ListenAndServe(":8040", hproxy)

	// Usage command line
	//
	// $ pip3 install httpie
	// $ UDID=$(idevice_id -l)
	// $ HTTP_PROXY=http://localhost:8040 http GET $UDID:8100/status
	//
	// Usage of python-lib: https://github.com/facebook/facebook-wda
	//
	// $ export HTTP_PROXY=http://localhost:8040
	// $ export DEVICE_URL=http://$UDID:8100
	// $ python
	// >>> import wda
	// >>> c = wda.Client()
	// >>> print(c.status())
	//
	// Usage of gwda: https://github.com/electricbubble/gwda
	//
	// $ export ...
	// c, _ := gwda.NewClient(os.Getenv("DEVICE_URL"))
	// c.IsWdaHealth()
}
