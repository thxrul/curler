# Maintainer: Arch Linux User <user@archlinux.org>
pkgname=curler
pkgver=1.0.0
pkgrel=1
pkgdesc="Arch Linux TUI download manager powered by curl continuous downloads"
arch=('x86_64' 'aarch64')
url="https://github.com/user/curler"
license=('MIT')
depends=('curl' 'glibc')
makedepends=('go')
optdepends=('libnotify: Desktop notifications via notify-send')
source=("${pkgname}-${pkgver}.tar.gz::https://github.com/user/${pkgname}/archive/v${pkgver}.tar.gz")
sha256sums=('SKIP')

build() {
  cd "$pkgname-$pkgver"
  export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
  export CGO_LDFLAGS="${LDFLAGS}"
  export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external"
  go build -o curler .
}

check() {
  cd "$pkgname-$pkgver"
  go test ./...
}

package() {
  cd "$pkgname-$pkgver"
  install -Dm755 curler "${pkgdir}/usr/bin/curler"
  install -Dm644 README.md "${pkgdir}/usr/share/doc/${pkgname}/README.md"
}
