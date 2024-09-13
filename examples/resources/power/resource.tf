resource "irmc-redfish_power" "pwr" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }


  //   |********************|*******************************************************************************************************|
  //   | IRMC Power options |                                           Description                                                 |
  //   |      (string)      |                                                                                                       |
  //   |--------------------|-------------------------------------------------------------------------------------------------------|
  //   | PushPowerButton    | Simulates a short power button press. The action depends on the power button configuration of the OS" |
  //   | On                 | Power on the platform.                                                                                |
  //   | ForceOn            | Immediate system power on.                                                                              |
  //   | GracefulRestart    | Performs a system shutdown and reboots the system.                                                    |
  //   | GracefulShutdown   | Performs a system shutdown and powers off the system.                                                 |
  //   | ForceRestart       | Immediate system reset without system shutdown.                                                       |
  //   | ForceOff           | Immediate system power off without system shutdown.                                                   |
  //   | PowerCycle         | Switches the system off and on again.                                                                 |
  //   | Nmi                | Triggers a (N)on-(M)askable (I)nterrupt (NMI) and halts the system.                                   |
  //   |********************|*******************************************************************************************************|


  host_power_action = "ForceRestart"

  max_wait_time = 150


}
