#!/usr/bin/perl -w

use strict;
use warnings;

# run me with 'while true; do { perl print.pl; } | nc -l 3000; done'

use Time::HiRes qw(usleep nanosleep);

my @devices = qw( ios xbox_one_s xbox_360 xbox_one );
my @titles = qw( daredevil cuervos narcos );
my @countries = qw( BR IND US RU );
my @successes = qw( success error );

my $first = 0;
while (1) {

    # print the response message the first time
    print "HTTP/1.1 200 OK\n\n" if $first++ == 0;

    my $time = time()*1000;

    printf('data: {"device":"%s","sev":"%s","title":"%s","country":"%s","time":%s}', $devices[int(rand(@devices))], $successes[rand(@successes)],$titles[int(rand(@titles))], $countries[int(rand(@countries))], $time);

    print "\n";
}

