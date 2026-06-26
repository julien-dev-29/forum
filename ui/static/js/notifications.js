(function() {
    if (typeof EventSource === 'undefined') return;

    var es = new EventSource('/api/notifications/stream');
    var els = document.querySelectorAll('.notif-count');

    es.onmessage = function(e) {
        try {
            var data = JSON.parse(e.data);
            if (data.count === undefined) return;
            var text = data.count > 0 ? '(' + data.count + ')' : '';
            for (var i = 0; i < els.length; i++) {
                els[i].textContent = text;
            }
        } catch(err) {}
    };
})();
