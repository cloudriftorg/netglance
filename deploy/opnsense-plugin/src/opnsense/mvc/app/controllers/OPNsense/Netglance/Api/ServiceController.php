<?php

namespace OPNsense\Netglance\Api;

use OPNsense\Base\ApiMutableServiceControllerBase;

class ServiceController extends ApiMutableServiceControllerBase
{
    protected static $internalServiceClass = '\OPNsense\Netglance\General';
    protected static $internalServiceTemplate = 'OPNsense/Netglance';
    protected static $internalServiceEnabled = 'enabled';
    protected static $internalServiceName = 'netglance';

    public function reconfigureAction()
    {
        if ($this->request->isPost()) {
            $this->sessionClose();
            // Re-render the env-file template from the config model, then
            // bounce the daemon so it re-reads NETGLANCE_* vars on startup.
            $backend = new \OPNsense\Core\Backend();
            $backend->configdRun('template reload OPNsense/Netglance');
            $mdl = new \OPNsense\Netglance\General();
            if ((string)$mdl->enabled === '1') {
                $backend->configdRun('netglance restart');
            } else {
                $backend->configdRun('netglance stop');
            }
            return ['status' => 'ok'];
        }
        return ['status' => 'failed'];
    }
}
