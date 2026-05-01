# Created by: netglance project
# $FreeBSD$

PORTNAME=	netglance
DISTVERSION=	0.1.0
CATEGORIES=	net-mgmt

MAINTAINER=	massimo.schiavo.p@gmail.com
COMMENT=	LAN host discovery and inventory daemon
WWW=		https://github.com/netglance/netglance

LICENSE=	MIT
LICENSE_FILE=	${WRKSRC}/LICENSE

# arp-scan does the actual L2 sweep; the daemon shells out to it.
RUN_DEPENDS=	arp-scan>=1.10:net-mgmt/arp-scan

# The release tarball produced by `make release-freebsd` already embeds the
# built React UI inside backend/internal/webui/dist, so the build host doesn't
# need npm — only Go.
USES=		go:modules

GO_PKGNAME=	github.com/netglance/netglance
GO_TARGET=	./backend/cmd/server:netglance
GO_BUILDFLAGS=	-trimpath -ldflags="-s -w -X main.version=${DISTVERSION}"

USE_RC_SUBR=	netglance

# Dedicated unprivileged user. arp-scan on FreeBSD is installed setuid root
# by net-mgmt/arp-scan, so the daemon can exec it from a non-root user.
USERS=		netglance
GROUPS=		netglance

PLIST_FILES=	sbin/netglance \
		"@dir(netglance,netglance,750) /var/db/netglance" \
		"@dir(netglance,netglance,755) etc/netglance"

post-install:
	${MKDIR} ${STAGEDIR}/var/db/netglance
	${MKDIR} ${STAGEDIR}${PREFIX}/etc/netglance

.include <bsd.port.mk>
