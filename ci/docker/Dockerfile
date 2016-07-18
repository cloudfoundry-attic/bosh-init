FROM bosh/main-ruby-go

ADD root_bashrc /root/.bashrc

# package manager provides 1.4.3, which is too old for vagrant-aws
ADD install-vagrant.sh /tmp/install-vagrant.sh
RUN /tmp/install-vagrant.sh && rm /tmp/install-vagrant.sh

# bosh-init dependencies
RUN apt-get install -y mercurial; apt-get clean

# lifecycle ssh test
RUN apt-get install -y sshpass; apt-get clean

# integration registry tests
RUN apt-get install -y openssh-server

# for hijack debugging
RUN apt-get install -y lsof psmisc strace
