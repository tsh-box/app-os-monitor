let memPlot = null;
let loadPlot = null;

$( document ).ready(function() {
    console.log( "ready!" );

    memPlot = $('#mem')[0];
	Plotly.plot( memPlot, [{
        x: [],
        y: [] }], 
        {margin: { t: 0 } },
        {staticPlot: true}
    );

    loadPlot = $('#load')[0];
	Plotly.plot( loadPlot, [{
        x: [],
        y: [] }], 
        { margin: { t: 0 } }, 
        {staticPlot: true}  
    );
    
    setInterval(updateData,1000);

});

function updateData() {
    $.getJSON('/app-os-monitor/ui/data',(data) => {
        console.log(data);
        var time = new Date();

        var mem = {
            x:  [[time]],
            y: [[data.freemem]]
        }

        var load = {
            x:  [[time]],
            y: [[data.loadavg1]]
        }

        var olderTime = time.setMinutes(time.getMinutes() - 1);
        var futureTime = time.setMinutes(time.getMinutes() + 1);

        var minuteView = {
                xaxis: {
                type: 'date',
                range: [olderTime,futureTime]
                }
            };

        Plotly.relayout(memPlot, minuteView);
        Plotly.extendTraces(memPlot, mem, [0])

        Plotly.relayout(loadPlot, minuteView);
        Plotly.extendTraces(loadPlot, load, [0])

    })
}