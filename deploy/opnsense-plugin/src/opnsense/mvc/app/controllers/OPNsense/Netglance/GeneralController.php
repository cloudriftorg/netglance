<?php

namespace OPNsense\Netglance;

use OPNsense\Base\IndexController as BaseIndexController;

/**
 * Renders the Services > Netglance tab. Mirrors the convention used
 * by the official OPNsense plugins (ntopng, wireguard, ...) where the
 * primary tab controller is named after the tab itself ("general") so
 * the menu URL /ui/<plugin>/general/index resolves cleanly.
 */
class GeneralController extends BaseIndexController
{
    public function indexAction()
    {
        $this->view->generalForm = $this->getForm('general');
        $this->view->pick('OPNsense/Netglance/general');
    }
}
