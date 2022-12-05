#!/bin/bash

sudo apt update && sudo apt install -y hashdeep fakeroot git wget
VERSION_FROM_GIT=$( git describe --tags `git rev-list --tags --max-count=1` )
VERSION=${VERSION_FROM_GIT:-0.0.1}
export VERSION=$VERSION.$GITHUB_RUN_NUMBER
export VERSION=${VERSION/v/}
DEBIAN_FRONTEND=noninteractive sudo apt install -y  rpm
mkdir install_linux/

mv DEBIAN/control TMP_control
mv DEBIAN/changelog TMP_changelog
mkdir -p vxagent/opt/pt/vxagent/{bin,logs,data} && \
	mkdir vxagent/DEBIAN && \
	cp _tmp/linux/386/vxagent vxagent/opt/pt/vxagent/bin && \
	mkdir -p vxagent/etc/systemd/system/ && \
	cp vxagent.service vxagent/etc/systemd/system/vxagent.service

arch="i386"
echo; echo "Generating vxagent/DEBIAN/control..."

eval "echo \"$(cat TMP_control)\"" > vxagent/DEBIAN/control
cat vxagent/DEBIAN/control || exit 1

echo; echo "Generating vxagent/DEBIAN/changelog..."
eval "echo \"$(cat TMP_changelog)\"" > vxagent/DEBIAN/changelog
cat vxagent/DEBIAN/changelog || exit 1

cp DEBIAN/* vxagent/DEBIAN

md5deep -r vxagent/opt/vxcontrol/vxagent > vxagent/DEBIAN/md5sums
chmod -R 755 vxagent/DEBIAN

fakeroot dpkg-deb --build vxagent vxagent-${VERSION}_${arch}.deb || exit 1

echo "Done create deb $arch"

rm -rf vxagent

mkdir -p vxagent/opt/pt/vxagent/{bin,logs,data} && \
	mkdir vxagent/DEBIAN && \
	cp _tmp/linux/amd64/vxagent vxagent/opt/pt/vxagent/bin && \
	mkdir -p vxagent/etc/systemd/system/ && \
	cp vxagent.service vxagent/etc/systemd/system/vxagent.service

arch="amd64"
echo; echo "Generating vxagent/DEBIAN/control..."

eval "echo \"$(cat TMP_control)\"" > vxagent/DEBIAN/control
cat vxagent/DEBIAN/control || exit 1

echo; echo "Generating vxagent/DEBIAN/changelog..."
eval "echo \"$(cat TMP_changelog)\"" > vxagent/DEBIAN/changelog
cat vxagent/DEBIAN/changelog || exit 1

cp DEBIAN/* vxagent/DEBIAN

md5deep -r vxagent/opt/vxcontrol/vxagent > vxagent/DEBIAN/md5sums
chmod -R 755 vxagent/DEBIAN

fakeroot dpkg-deb --build vxagent vxagent-${VERSION}_${arch}.deb || exit 1

echo "Done create deb $arch"

cp *.deb install_linux/
export VXSERVER_CONNECT="VXSERVER_CONNECT"
rm -rf ~/rpmbuild/SOURCES/* || true
mkdir -p ~/rpmbuild/SOURCES/vxagent/{bin,unit}

arch="386"
eval "echo \"$(cat RPM/rpm.spec)\"" > rpm_$arch.spec
cp _tmp/linux/386/vxagent ~/rpmbuild/SOURCES/vxagent/bin/
cp vxagent.service ~/rpmbuild/SOURCES/vxagent/unit/

rpmbuild -bb ./rpm_$arch.spec --target i386
cp ~/rpmbuild/RPMS/i386/* install_linux/vxagent-${VERSION}_i386.rpm

arch="amd64"
rm -rf ~/rpmbuild/SOURCES/* || true
mkdir -p ~/rpmbuild/SOURCES/vxagent/{bin,unit}
cp _tmp/linux/amd64/vxagent ~/rpmbuild/SOURCES/vxagent/bin/
cp vxagent.service ~/rpmbuild/SOURCES/vxagent/unit/

eval "echo \"$(cat RPM/rpm.spec)\"" > rpm_$arch.spec
rpmbuild -bb ./rpm_$arch.spec --target amd64
cp ~/rpmbuild/RPMS/amd64/* install_linux/vxagent-${VERSION}_amd64.rpm

