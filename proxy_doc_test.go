package mlx

import "fmt"

func ExampleBuildProfileProxyFromGenerated() {
	conn, _ := ParseGeneratedProxyConnection(
		"gate.multilogin.com:1080:2235470499_bc98e4f8_multilogin_com-country-us-region-new_jersey-city-east_brunswick-sid-demo:secret",
		ProxyProtocolSOCKS5,
	)
	proxy := BuildProfileProxyFromGenerated(conn)

	fmt.Println(proxy.Type)
	fmt.Println(proxy.Country, proxy.Region, proxy.City)
	// Output:
	// socks5
	// us new_jersey east_brunswick
}
