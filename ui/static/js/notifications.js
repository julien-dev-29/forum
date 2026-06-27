(function() {
    if (typeof EventSource === 'undefined') return;

    var es = new EventSource('/api/notifications/stream');
    var els = document.querySelectorAll('.notif-badge');

    es.onmessage = function(e) {
        try {
            var data = JSON.parse(e.data);
            if (data.count === undefined) return;
            for (var i = 0; i < els.length; i++) {
                if (data.count > 0) {
                    els[i].textContent = data.count;
                    els[i].style.display = '';
                } else {
                    els[i].style.display = 'none';
                }
            }
        } catch(err) {}
    };
})();
