import {
    for_each = var.rack1
    id = "echo ${"prop":"val"}"
    to = irmc-redfish_virtual_media.vm[each.key]
}
