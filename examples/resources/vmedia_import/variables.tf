variable "rack1" {
    type = map(object({
        username = string
        password = string
        endpoint = string
        ssl_insecure = bool
    }))
}
