Name:           vxagent
Version:        $VERSION
Release:        $GITHUB_RUN_NUMBER
Summary:        This service for work XDR agent
License:        -

Source0:        vxagent
Source1:        libraries.tar.gz
Requires:       bash, systemd, initscripts, libstdc++, libgcc, glibc, gettext

BuildRoot:      %(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)


%description
VX agent

%global _enable_debug_package 0
%global debug_package %{nil}
%global __os_install_post /usr/lib/rpm/brp-compress %{nil}

%install
export DONT_STRIP=1
mkdir -p %{buildroot}/opt/vxcontrol/vxagent/bin
mkdir -p %{buildroot}/opt/vxcontrol/vxagent/logs
mkdir -p %{buildroot}/opt/vxcontrol/vxagent/data
mkdir -p %{buildroot}/etc/systemd/system
install -D -pm 755 %{SOURCE0}/bin/vxagent %{buildroot}/opt/vxcontrol/vxagent/bin/vxagent
install -D -pm 755 %{SOURCE0}/unit/vxagent.service %{buildroot}/etc/systemd/system/vxagent.service
mkdir -p %{buildroot}/usr/lib/%{name}
tar -xzf %{SOURCE1} -C %{buildroot}/usr/lib/%{name}/


%post -p /bin/bash
ln -s /usr/lib64/librt.so.1 /usr/lib64/librt.so 2>/dev/null || true
ln -s /usr/lib64/libpthread.so.0 /usr/lib64/libpthread.so 2>/dev/null || true
chmod +x /opt/vxcontrol/vxagent/bin/vxagent && chown root:root /opt/vxcontrol/vxagent/bin/vxagent

tmpfile=$(mktemp)
VXSERVER_CONNECT="${VXSERVER_CONNECT:-'wss://localhost:8443'}" envsubst < /etc/systemd/system/vxagent.service > ${tmpfile}
cat ${tmpfile} > /etc/systemd/system/vxagent.service
rm -f ${tmpfile}
systemctl daemon-reload

/opt/vxcontrol/vxagent/bin/vxagent -command start

%preun -p /bin/bash
/opt/vxcontrol/vxagent/bin/vxagent -command stop || true
if pgrep -f vxagent &>/dev/null; then
  kill -9 $(cat /var/run/vxagent.pid 2>/dev/null) &>/dev/null || true
fi
/opt/vxcontrol/vxagent/bin/vxagent -command uninstall || true

%postun -p /bin/bash
if ! [ -d /opt/ ]; then
  mkdir "/opt" || true
fi
rm -rf /opt/vxcontrol/vxagent || true
rm -rf /usr/lib/vxagent || true
rmdir /opt/vxcontrol/ || true

%files
/opt/vxcontrol/vxagent/*
/etc/systemd/system/*
/usr/lib/vxagent/*

%clean
rm -rf $RPM_BUILD_ROOT

%changelog
* Mon Nov 22 2021 VXControl <help@vxcontrol.com>
- Create RPM build
