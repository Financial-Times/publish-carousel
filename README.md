# Publish Carousel

A microservice that continuously republishes content and annotations from the native store.

# Developer Notes

The Publish Carousel writes a metadata file to S3 on a graceful shutdown. Unfortunately, this functionality does not work on Windows using Git Bash, but does work when using the Command Prompt. It works as expected, however, on a Mac.
