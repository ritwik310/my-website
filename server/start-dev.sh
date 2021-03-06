#!/bin/bash

# colors holds variables responsible for stdout colors
COLOR_WARNING="\e[33m\e[1m" # for warning
COLOR_INFO="\e[36m\e[1m" # for info
COLOR_NORMAL="\e[0m"        # for normal (for making text normal again)

# environment variables for developemnt
export DEV_MODE="true"
export MONGO_URI="mongodb://localhost:27017"
export DB_NAME="my-website-prod"
export TEST_DB_NAME="my-website-test"
export GOOGLE_AUTH_REDIRECT_URL="http://localhost:8080/api/auth/google/callback"

# reading ./secrets.config file, that includes secrets formatted as the following
# SECRET_ENV_1=something;
# SECRET_ENV_2=something_more;
# SECRET_ENV_3=something_more_more;

# reading the contents of file
SECRETS=$(<dev-keys.secret)
# replacing `\n` from SECRETS string
SECRETS=$(echo $SECRETS|tr -d '\n')
# adding a ` ` at the end of the string, so later can be splitted at `; `
SECRETS=$(printf "${SECRETS} ")

# splitting teh string from `; `
IFS='; ' read -ra EACH <<< "${SECRETS}"
# for each chunk
for i in "${EACH[@]}"; do
    # again splitting at `=`
    IFS='=' read -ra PART <<< "$i"
    # eventually exporting the environment variable
    export "${PART[0]}"="${PART[1]}"
done

# secret environment variables that are expected to be declared
declare -a secrets=("SESSION_KEY" "GOOGLE_CLIENT_ID" "GOOGLE_CLIENT_SECRET" "AUTHORIZED_EMAIL_I" "AUTHORIZED_EMAIL_II" "ADMIN_ORIGIN")

# checking if secret config variables exists or not
for i in "${secrets[@]}"
do
  if [[ ! -v "$i" ]]; then
    # warning if some secret variable is not present in the environment
    echo -e "${COLOR_WARNING}WARN${COLOR_NORMAL} environment variable not found - \"${i}\""
  else
    :
  fi
done


# starting the go server
echo -e "${COLOR_WARNING}INFO${COLOR_NORMAL} starting the server...\n"
go run main.go
