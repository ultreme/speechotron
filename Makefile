GOPKG ?=	ultre.me/speechotron
DOCKER_IMAGE ?=	moul/speechotron
GOBINS ?=	.
NPM_PACKAGES ?=	.

all: test install

-include rules.mk
