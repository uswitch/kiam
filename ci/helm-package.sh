#!/usr/bin/env sh

set -e

# install git
apk add git

# check if changes have been made to the helm chart
if ! git diff HEAD~1 | grep -iE 'a\/helm.*';
then
  echo "No changes to helm chart made, skipping..."
  exit 0
else
  # package helm chart
  helm init --stable-repo-url https://charts.helm.sh/stable --client-only
  mkdir ${DRONE_WORKSPACE}/output/
  helm package ${DRONE_WORKSPACE}/helm/${APP}/ -d ${DRONE_WORKSPACE}/output/

  # create new git repo and add remote
  mkdir ${DRONE_WORKSPACE}/new-repo/ && cd ${DRONE_WORKSPACE}/new-repo/
  git init
  git config --global user.email ${CI_EMAIL}
  git remote add origin https://${DRONE_NETRC_USERNAME}:${DRONE_NETRC_PASSWORD}@${REPO}
  git fetch
  git checkout --track origin/gh-pages
  git pull

  # index new chart and merge old index to preserve chart creation dates
  helm repo index ${DRONE_WORKSPACE}/output/ --merge ${DRONE_WORKSPACE}/new-repo/charts/index.yaml
  mv ${DRONE_WORKSPACE}/output/* ${DRONE_WORKSPACE}/new-repo/charts/

  # stage and commit new files, push to remote
  git add .
  git commit -m "Original commit: ${DRONE_COMMIT_SHA}"
  git push -u origin gh-pages
fi
