VersionTag="$(git describe --always --exact-match --tags $(git log -n1 --pretty='%h'))"
Repo="vwxyzjn/portwarden-server-prod"
docker version

if [ -z "$VersionTag" ]  # If git version tag is empty
then
    echo "tag is empty"
    docker build -t "$Repo:latest" -f Dockerfile.Build --build-arg Salt=$Salt .
    echo "build successful"
    docker push "$Repo:latest"
    echo "push successful"
else
    echo "tag exists"
    docker build -t "$Repo:$VersionTag" -t "$Repo:latest" -f Dockerfile.Build --build-arg Salt=$Salt .
    echo "build successful"
    docker push "$Repo:$VersionTag"
    docker push "$Repo:latest"
    echo "push successful"
fi
