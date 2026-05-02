<?php

namespace OPNsense\Netglance\Api;

use OPNsense\Base\ApiMutableServiceControllerBase;

/**
 * Service control endpoints (start/stop/restart/reconfigure/status) for
 * the netglance daemon. The base class implements every action — we only
 * need to wire up the four $internal* properties.
 */
class ServiceController extends ApiMutableServiceControllerBase
{
    protected static $internalServiceClass = '\OPNsense\Netglance\General';
    protected static $internalServiceTemplate = 'OPNsense/Netglance';
    protected static $internalServiceEnabled = 'enabled';
    protected static $internalServiceName = 'netglance';
}
