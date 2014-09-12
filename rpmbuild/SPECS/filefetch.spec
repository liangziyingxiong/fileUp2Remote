Name:		filefetch
Version:	1.0
Release:	3%{?dist}
Summary:	srs log file up to remote server through sftp

Group:		system exec file
License:	Copyright on @maichuang
URL:		http://www.maichuang.net
Source0:	filefetch.tar.gz
BuildRoot:	%(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)
Packager:       xiaoliang.hao<liangziyingxiong@gmail.com>
BuildRequires:	go
#Requires:	

%description
srs log file up to remote server through sftp

%prep
mkdir -p %{buildroot}/usr/sbin/
mkdir -p %{buildroot}/etc/cron.d/
mkdir -p %{buildroot}/usr/local/srs/conf/
echo package "%{name}-%{version}-%{release}" begin installing


%setup -c

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}/usr/sbin/
mkdir -p %{buildroot}/etc/cron.d/
mkdir -p %{buildroot}/usr/local/srs/conf/
install -m 755 filefetch       %{buildroot}/usr/sbin/
install -m 644 srslogrotate    %{buildroot}/usr/local/srs/conf/
install -m 644 srslogcron      %{buildroot}/usr/local/srs/conf/
install -m 644 filefetch_cron  %{buildroot}/usr/local/srs/conf/

%clean
rm -rf %{buildroot}

%post
sed -i /"\/usr\/sbin\/filefetch"/d /etc/crontab
sed -i /"\/usr\/sbin\/logrotate \/usr\/local\/srs\/conf\/srslogrotate"/d /etc/crontab
cat /usr/local/srs/conf/filefetch_cron >> /etc/crontab
cat /usr/local/srs/conf/srslogcron     >> /etc/crontab
echo package "%{name}-%{version}-%{release}" installed successfully

%files
%defattr(-,root,root,-)
/usr/sbin/filefetch
/usr/local/srs/conf/srslogrotate
/usr/local/srs/conf/filefetch_cron
/usr/local/srs/conf/srslogcron
#/etc/crontab
#/etc/cron.d/filefetch_cron
#/etc/cron.d/srslogcron

%changelog
* Mon Sep 1 2014 xiaoliang.hao<liangziyingxiong@gmail.com>
- file package

%postun
sed -i /"\/usr\/sbin\/filefetch"/d /etc/crontab
sed -i /"\/usr\/sbin\/logrotate \/usr\/local\/srs\/conf\/srslogrotate"/d /etc/crontab
echo package "%{name}-%{version}-%{release}" uninstalled successfully
