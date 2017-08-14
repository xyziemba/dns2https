#!/usr/bin/env bats

@test "correctly returns A for xyziemba.com" {
    reference="$(dig \@8.8.8.8 +short xyziemba.com | sort)"
    test="$(dig \@localhost -p $PORT +short xyziemba.com | sort)"
    testTcp="$(dig \@localhost -p $PORT +short +tcp xyziemba.com | sort)"
    [ "$reference" == "$test" ]
    [ "$reference" == "$testTcp" ]
}

@test "correctly returns MX for xyziemba.com" {
    reference="$(dig \@8.8.8.8 +short MX xyziemba.com | sort)"
    test="$(dig \@localhost -p $PORT +short MX xyziemba.com | sort)"
    testTcp="$(dig \@localhost -p $PORT +short +tcp MX xyziemba.com | sort)"
    [ "$reference" == "$test" ]
    [ "$reference" == "$testTcp" ]
}

@test "correctly doesn't return A for dnssec.fail" {
    reference="$(dig \@8.8.8.8 +short dnssec.fail | sort)"
    test="$(dig \@localhost -p $PORT +short dnssec.fail | sort)"
    testTcp="$(dig \@localhost -p $PORT +short +tcp dnssec.fail | sort)"
    [ "$reference" == "$test" ]
    [ "$reference" == "$testTcp" ]
}

@test "correctly returns A for dnssec.fail with CD bit checked" {
    reference="$(dig \@8.8.8.8 +short +cd dnssec.fail | sort)"
    test="$(dig \@localhost -p $PORT +short +cd dnssec.fail | sort)"
    testTcp="$(dig \@localhost -p $PORT +short +cd +tcp dnssec.fail | sort)"
    [ "$reference" == "$test" ]
    [ "$reference" == "$testTcp" ]
}