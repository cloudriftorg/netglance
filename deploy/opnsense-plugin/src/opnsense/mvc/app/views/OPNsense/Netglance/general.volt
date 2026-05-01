<script>
$(document).ready(function () {
    var data_get_map = {'frm_general': '/api/netglance/general/get'};
    mapDataToFormUI(data_get_map).done(function (data) {
        formatTokenizersUI();
        $('.selectpicker').selectpicker('refresh');
    });

    updateServiceControlUI('netglance');

    $("#saveAct").click(function () {
        saveFormToEndpoint('/api/netglance/general/set', 'frm_general', function () {
            ajaxCall('/api/netglance/service/reconfigure', {}, function (data, status) {
                updateServiceControlUI('netglance');
            });
        });
    });

    $("#openUI").click(function () {
        var port = $("#general\\.httpPort").val() || '8473';
        var addr = window.location.hostname;
        // bindAddress 0.0.0.0 means "listen everywhere" — open the UI on the
        // same hostname the user is browsing OPNsense from. If the user set a
        // specific bind address, use that.
        var bind = $("#general\\.bindAddress").val();
        if (bind && bind !== '0.0.0.0' && bind !== '::') {
            addr = bind;
        }
        window.open('http://' + addr + ':' + port + '/', '_blank');
    });
});
</script>

<ul class="nav nav-tabs" data-tabs="tabs" id="maintabs">
    <li class="active">
        <a data-toggle="tab" href="#general">{{ lang._('General') }}</a>
    </li>
</ul>

<div class="content-box tab-content">
    <div id="general" class="tab-pane fade in active">
        {{ partial("layout_partials/base_form", ['fields': generalForm, 'id': 'frm_general']) }}
    </div>
</div>

<div class="content-box" style="padding-bottom: 1.5em;">
    <div class="col-md-12">
        <hr/>
        <button class="btn btn-primary" id="saveAct" type="button">
            <b>{{ lang._('Save') }}</b>
            <i id="saveAct_progress" class=""></i>
        </button>
        <button class="btn btn-default" id="openUI" type="button">
            <i class="fa fa-external-link"></i>
            {{ lang._('Open Netglance UI') }}
        </button>
    </div>
</div>
