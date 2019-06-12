<template>
    <div style="margin-left: auto; margin-right: auto; margin-top: 10px; margin-bottom: 80px; width: 70%; ">
        <div ref="map" id="map" style="width: 100%; height: 600px;"></div>
        <table class="table is-bordered" style="width: 96%; margin-left: auto; margin-right: auto">
            <thead>
            <tr>
                <th>Node Ip</th>
                <th>Sync Port</th>
                <th>Height</th>
                <th>Node Type</th>
                <th>Country</th>
                <th>Connect</th>
                <th>Latest Active Time(GMT)</th>
                <th>Software Version</th>
            </tr>
            </thead>
            <transition-group name="flip-list" tag="tbody">
                <tr v-for="node in nodes" v-bind:key="node.ip">
                    <td style="width: 180px;">{{ node.ip }}</td>
                    <td style="width: 80px;">{{ node.port }}</td>
                    <td style="width: 100px; min-width: 100px;">{{ node.height }}</td>
                    <td style="width: 100px;">{{ node.services === 1 ? 'Verify' : 'Service' }}</td>
                    <td style="width: 100px; min-width: 100px">{{ node.country }}</td>
                    <td style="width: 100px;">{{ node.can_connect }}</td>
                    <td style="width: 200px; min-width: 120px">{{ fmtTime(node.last_active_time) }}</td>
                    <td style="width: 200px">{{ node.soft_version }}</td>
                </tr>
            </transition-group>
        </table>
        <footer style="margin-top: 15px;">
            <h3>About Ontology Node Map</h3>
            <p>Ontology is currently being developed to estimate the size of the Ontology network by finding all the
                reachable nodes in the network. The current methodology involves sending version messages recursively to
                find all the reachable nodes in the network, starting from a set of seed nodes</p>
            <a @click="onClickApiDoc">API Doc</a>
        </footer>
        <div class="api" v-show="showApi">
            <hr>
            <h3>Node Map Api</h3>
            <br>
            <h4><a href="#list-nodes">List Nodes</a></h4>
            <pre> GET /api/nodes</pre>
            <p>Example: </p>
            <pre>
$ curl -H "Accept: application/json; indent=4" http://ont-node-map.woshifyz.com/api/nodes

[
    {
        ip: "104.45.19.210",
        port: 20338,
        services: 2,
        height: 4635009,
        is_consensus: false,
        soft_version: "v1.6.2-0-g2702656",
        is_http: false,
        http_info_port: 0,
        consensus_port: 0,
        last_active_time: 1559967635886,
        can_connect: true,
        lat: 52.3702,
        lon: 4.89517,
        country: "Netherlands"
    },
    .
    .
    {
        ip: "13.112.52.91",
        port: 20338,
        services: 2,
        height: 0,
        is_consensus: false,
        soft_version: "",
        is_http: false,
        http_info_port: 0,
        consensus_port: 0,
        last_active_time: 1559967532535651600,
        can_connect: false,
        lat: 35.709,
        lon: 139.732,
        country: "Japan"
    }
]
            </pre>
        </div>
    </div>
</template>

<script>

  import * as am4core from "@amcharts/amcharts4/core";
  import * as am4maps from "@amcharts/amcharts4/maps";
  import am4themes_animated from "@amcharts/amcharts4/themes/animated";
  import am4geodata_worldLow from "@amcharts/amcharts4-geodata/worldLow"
  import axios from "axios"

  am4core.useTheme(am4themes_animated);

  export default {
    name: 'app',
    components: {},
    data() {
      return {
        nodes: [],
        showApi: false,
      }
    },
    methods: {
      fmtTime: function (ts) {
        if (ts > 10000000000000) {
          ts /= 1000000
        }
        return new Date(ts).toISOString().slice(0, 19).replace('T', ' ')
      },
      onClickApiDoc: function (e) {
        e.preventDefault();
        this.showApi = !this.showApi;
      },
      dataToNodes: function (data) {
        var nodes = [];
        for (var i = 0; i < data.length; i++) {
          var node = data[i];
          var key = node.ip + ":" + node.port;
          if (node.lat > 900 || node.height === -1) {
            continue
          }
          var color = this.chart.colors.getIndex(2);
          if (node.services === 1) {
            color = this.chart.colors.getIndex(1);
          }
          if (node.height === 0) {
            color = this.chart.colors.getIndex(0);
          }
          Object.assign(node,
            {
              "id": key,
              "latitude": node.lat,
              "longitude": node.lon,
              "name": key,
              "value": node.height,
              "color": color
            }
          );
          nodes.push(node)
        }
        return nodes;
      },
      loadNodes: function (cb) {
        var self = this;
        var host = "";
        // host = "http://localhost:8888";
        axios.get(host + "/api/nodes")
          .then(function (response) {
            self.nodes = self.dataToNodes(response.data);
            if (cb) {
              cb(self.nodes);
            }
          })
      }
    },
    mounted() {
      var chart = am4core.create("map", am4maps.MapChart);

      chart.geodata = am4geodata_worldLow;

      chart.projection = new am4maps.projections.Miller();
      chart.maxZoomLevel = 1;

      var polygonSeries = chart.series.push(new am4maps.MapPolygonSeries());
      polygonSeries.exclude = ["AQ"];
      polygonSeries.useGeodata = true;
      polygonSeries.nonScalingStroke = true;
      polygonSeries.strokeWidth = 0.5;

      this.loadNodes(function (nodes) {
        var imageSeries = chart.series.push(new am4maps.MapImageSeries());
        imageSeries.data = nodes;
        imageSeries.dataFields.value = "value";

        var imageTemplate = imageSeries.mapImages.template;
        imageTemplate.propertyFields.latitude = "latitude";
        imageTemplate.propertyFields.longitude = "longitude";
        imageTemplate.nonScaling = true;

        var circle = imageTemplate.createChild(am4core.Circle);
        circle.fillOpacity = 0.4;
        circle.propertyFields.fill = "color";
        circle.tooltipText = "{name}: [bold]{value}[/]";

        imageSeries.heatRules.push({
          "target": circle,
          "property": "radius",
          "min": 5,
          "max": 15,
          "dataField": "value"
        });
      });
      this.chart = chart;

      var self = this;
      setInterval(function () {
        self.loadNodes(null);
      }, 6000);
    },

    beforeDestroy() {
      if (this.chart) {
        this.chart.dispose();
      }
    }
  }
</script>

<style>
    html {
        -webkit-font-smoothing: antialiased;
    }

    body {
        background: #fff;
        color: #333;
        font-family: play, sans-serif;
        font-size: 13px;
        line-height: 1.5;
    }

    a {
        color: #3273dc;
        cursor: pointer;
        text-decoration: none;
    }

    h1, h2, h3, h4 {
        font-family: play, sans-serif;
        padding: 0;
        margin: 0;
        text-transform: uppercase;
    }

    h3, h4 {
        font-size: 1.2em;
        font-weight: 700;
    }

    p {
        display: block;
        margin-block-start: 1em;
        margin-block-end: 1em;
        margin-inline-start: 0px;
        margin-inline-end: 0px;
    }

    *, ::after, ::before {
        box-sizing: inherit;
    }

    .table {
        background-color: #fff;
        color: #363636;
        border-collapse: collapse;
        border-spacing: 0;
        display: table;
    }

    .table td, .table th {
        border: 1px solid #dbdbdb;
        padding: .5em .75em;
        vertical-align: top;
    }

    .table.is-bordered tr:last-child td, .table.is-bordered tr:last-child th {
        border-bottom-width: 1px;
    }

    .table.is-bordered td, .table.is-bordered th {
        border-width: 1px;
    }

    .table th:not([align]) {
        text-align: left;
    }

    hr {
        background-color: #f5f5f5;
        border: none;
        display: block;
        height: 2px;
        margin: 1.5rem 0;
    }

    pre {
        -webkit-overflow-scrolling: touch;
        background-color: #f5f5f5;
        color: #4a4a4a;
        font-size: .875em;
        overflow-x: auto;
        padding: 1.25rem 1.5rem;
        white-space: pre;
        word-wrap: normal;
    }

    .flip-list-move {
        transition: transform 0.5s;
    }
</style>
