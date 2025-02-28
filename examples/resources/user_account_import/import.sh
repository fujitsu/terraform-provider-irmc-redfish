#
# Copyright (c) 2024 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.
# 
# Licensed under the Mozilla Public License Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://mozilla.org/MPL/2.0/
# 
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#!/bin/bash

# This script uses information about user accounts from terraform.tfvars to import user data from the Redfish server.
# Each user must have information about their ID and username. Using this information, the script creates Terraform requests to
# import user account data and prepares it for further modification.


SERVER_USERNAME="admin"
SERVER_PASSWORD="admin"
SERVER_ENDPOINT="https://10.172.201.36"
SSL_INSECURE=true

declare -A USERS

while IFS= read -r line; do
  if [[ $line =~ ^[[:space:]]*\"([^\"]+)\"[[:space:]]*=[[:space:]]*\{ ]]; then
    USER_KEY="${BASH_REMATCH[1]}"
    USER_ID=""
    USERNAME=""

    while IFS= read -r line && [[ ! $line =~ ^[[:space:]]*\} ]]; do
      if [[ $line =~ user_id[[:space:]]*=[[:space:]]*\"([^\"]+)\" ]]; then
        USER_ID="${BASH_REMATCH[1]}"
      elif [[ $line =~ username[[:space:]]*=[[:space:]]*\"([^\"]+)\" ]]; then
        USERNAME="${BASH_REMATCH[1]}"
      fi
    done

    USERS["$USER_KEY"]="$USER_ID:$USERNAME"
  fi
done < terraform.tfvars

for USER_KEY in "${!USERS[@]}"; do
  IFS=":" read -r USER_ID USERNAME <<< "${USERS[$USER_KEY]}"

  if [[ -n "$USER_ID" && -n "$USERNAME" ]]; then
    terraform import "irmc-redfish_user_account.ua[\"$USER_KEY\"]" "{
      \"username\": \"$SERVER_USERNAME\",
      \"password\": \"$SERVER_PASSWORD\",
      \"endpoint\": \"$SERVER_ENDPOINT\",
      \"ssl_insecure\": $SSL_INSECURE,
      \"user_id\": \"$USER_ID\",
      \"user_username\": \"$USERNAME\"
    }"

    if [ $? -eq 0 ]; then
      echo "Successfully imported user $USER_KEY with ID $USER_ID"
    else
      echo "Error importing user $USER_KEY with ID $USER_ID"
    fi
  fi
done
