<?php

/*
 * Plugin hooks for OPNsense's syshook system: registers the netglance
 * daemon as a known service so it appears under Services > Status and
 * can be controlled via the standard service widget.
 */

function netglance_services()
{
    $services = [];

    $mdl = new \OPNsense\Netglance\General();
    if ((string)$mdl->enabled === '1') {
        $services[] = [
            'description' => gettext('netglance — LAN host discovery'),
            'configd' => [
                'restart' => ['netglance restart'],
                'start' => ['netglance start'],
                'stop' => ['netglance stop'],
            ],
            'name' => 'netglance',
            'pidfile' => '/var/run/netglance.pid',
        ];
    }

    return $services;
}
