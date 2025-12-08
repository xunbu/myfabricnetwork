export const formatNumber = n => n ? n.toLocaleString() : 0;

export const formatTimestamp = t => t ? new Date(t).toLocaleString('zh-CN') : '-';

export const formatBytes = b => {
    if (!b) return '0 B';
    const i = Math.floor(Math.log(b) / Math.log(1024));
    return `${(b / Math.pow(1024, i)).toFixed(2)} ${['B', 'KB', 'MB'][i]}`;
};

export const truncateText = (t, l) => t && t.length > l ? t.substring(0, l) + '...' : t;

export const formatTxType = (type) => {
    const map = {
        'ENDORSER_TRANSACTION': '背书交易',
        'CONFIG': '配置交易',
        'CONFIG_UPDATE': '配置更新',
        'PEER_RESOURCE_UPDATE': '节点资源更新'
    };
    return map[type] || type;
};

export const getUsageColor = v => v > 80 ? 'text-danger' : (v > 50 ? 'text-warning' : 'text-success');