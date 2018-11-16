FROM plugins/base:multiarch
ADD release/linux/amd64/drone-chef-client /bin/
ENTRYPOINT ["/bin/drone-chef-client"]

