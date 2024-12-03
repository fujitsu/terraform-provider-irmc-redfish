/*
Copyright (c) 2024 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

servers = {
  "rack1" = {
    username     = "admin"
    password     = "admin"
    endpoint     = "https://10.172.201.36"
    ssl_insecure = true
  }
}

# list of users to import and modify
# user_id and username are required 
# rest options are optional to set it per user
users = {
  "User_1" = {
    user_id   = "4"
    username  = "Test_P"
    user_role = "Operator"
  },
  "User_2" = {
    user_id  = "5"
    username = "Test_D"
  },
  "User_3" = {
    user_id  = "6"
    username = "Test_H"
  },
  "User_4" = {
    user_id                     = "3"
    username                    = "Test_XX"
    user_remote_storage_enabled = false
  }
}
