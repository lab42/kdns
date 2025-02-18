package handler

const (
	mdnsDefaultType   = "_http._tcp"
	mdnsDefaultDomain = "local"
	mdnsDefaultPort   = 80

	mdnsEnabledAnnotation = "lab42.io/mdns.enabled"
	mdnsNameAnnotation    = "lab42.io/mdns.name"
	mdnsTypeAnnotation    = "lab42.io/mdns.type"
	mdnsDomainAnnotation  = "lab42.io/mdns.domain"
	mdnsHostAnnotation    = "lab42.io/mdns.host"
	mdnsTextAnnotation    = "lab42.io/mdns.text"
	mdnsIPsAnnotation     = "lab42.io/mdns.ip"
	mdnsPortAnnotation    = "lab42.io/mdns.port"
	mdnsIfacesAnnotation  = "lab42.io/mdns.Ifaces"
)
