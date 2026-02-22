<div class="row">
    <div class="col-md-12">
        <div class="panel panel-default">
            <div class="panel-heading">
                <h3 class="panel-title">FLVX 流量转发服务</h3>
            </div>
            <div class="panel-body">
                {if $status eq 'active'}
                <div class="row">
                    <div class="col-md-6">
                        <div class="panel panel-info">
                            <div class="panel-heading">
                                <h4 class="panel-title">账户信息</h4>
                            </div>
                            <div class="panel-body">
                                <table class="table table-bordered">
                                    <tr>
                                        <td><strong>用户名</strong></td>
                                        <td>{$username}</td>
                                    </tr>
                                    <tr>
                                        <td><strong>密码</strong></td>
                                        <td>
                                            <code id="password-display">{$password}</code>
                                            <button class="btn btn-xs btn-default" onclick="togglePassword()">
                                                <i class="fa fa-eye"></i>
                                            </button>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td><strong>隧道</strong></td>
                                        <td>{$tunnelName} (ID: {$tunnelId})</td>
                                    </tr>
                                    <tr>
                                        <td><strong>最大转发数</strong></td>
                                        <td>{$maxForwards}</td>
                                    </tr>
                                </table>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-6">
                        <div class="panel panel-info">
                            <div class="panel-heading">
                                <h4 class="panel-title">流量使用</h4>
                            </div>
                            <div class="panel-body">
                                <div class="progress" style="height: 25px; margin-bottom: 15px;">
                                    <div class="progress-bar {if $trafficPercentage gt 80}progress-bar-danger{elseif $trafficPercentage gt 50}progress-bar-warning{else}progress-bar-success{/if}" 
                                         role="progressbar" 
                                         aria-valuenow="{$trafficPercentage}" 
                                         aria-valuemin="0" 
                                         aria-valuemax="100" 
                                         style="width: {$trafficPercentage}%;">
                                        {$trafficPercentage}%
                                    </div>
                                </div>
                                <table class="table table-bordered">
                                    <tr>
                                        <td><strong>已用流量</strong></td>
                                        <td>{$usedTraffic}</td>
                                    </tr>
                                    <tr>
                                        <td><strong>总流量</strong></td>
                                        <td>{$totalTraffic}</td>
                                    </tr>
                                    <tr>
                                        <td><strong>到期时间</strong></td>
                                        <td>{$expiryDate}</td>
                                    </tr>
                                </table>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div class="row" style="margin-top: 20px;">
                    <div class="col-md-12">
                        <div class="panel panel-default">
                            <div class="panel-heading">
                                <h4 class="panel-title">快速操作</h4>
                            </div>
                            <div class="panel-body">
                                <a href="{$panelUrl}" target="_blank" class="btn btn-primary">
                                    <i class="fa fa-external-link"></i> 打开 FLVX 面板
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
                
                {elseif $status eq 'not_provisioned'}
                <div class="alert alert-warning">
                    <i class="fa fa-exclamation-triangle"></i> 
                    服务尚未配置，请等待系统处理或联系客服。
                </div>
                
                {elseif $status eq 'error'}
                <div class="alert alert-danger">
                    <i class="fa fa-times-circle"></i> 
                    获取服务信息时出错: {$error}
                </div>
                {/if}
            </div>
        </div>
    </div>
</div>

<script>
function togglePassword() {
    var el = document.getElementById('password-display');
    if (el.style.filter === 'blur(5px)') {
        el.style.filter = 'none';
    } else {
        el.style.filter = 'blur(5px)';
    }
}
// Initially blur password (only if element exists)
var pwdEl = document.getElementById('password-display'); if (pwdEl) { pwdEl.style.filter = 'blur(5px)'; }
</script>
