

tar xvf mysql-5.7.17.tar.gz
cd mysql-5.7.17
cmake . -DCMAKE_INSTALL_PREFIX=/usr/local/mysql -DMYSQL_DATADIR=/usr/local/mysql/data -DDEFAULT_CHARSET=utf8 -DDEFAULT_COLLATION=utf8_general_ci -DMYSQL_TCP_PORT=3306 -DMYSQL_USER=mysql -DWITH_BOOST=/usr/local/boost/boost_1_59_0
make
make install






















