function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

let envoy_cm = null;
let curl_cm = null;

const app = Vue.createApp({
    data() {
        return {
            envoy_config: "",
            curl_command: "",
            curl_result: "",
            default_configs: {},
            error: false,
            loading: false,
            copied: false,
        }
    },
    mounted: async function() {
        // load state
        const state = this.loadFromHash() || this.loadFromLocalStorage() || (await this.loadDefault());
        this.envoy_config = state.envoy_config;
        this.curl_command = state.curl_command;

        // create envoy codemirror
        const envoyArea = document.querySelector('#envoy');
        envoyArea.innerHTML = this.envoy_config;
        envoy_cm = CodeMirror.fromTextArea(envoyArea, {
            lineNumbers: true,
            mode: 'yaml',
        });
        envoy_cm.setSize('100%', '100%');

        // create curl codemirror
        const curlArea = document.querySelector('#curl');
        curlArea.innerHTML = this.curl_command;
        curl_cm = CodeMirror.fromTextArea(curlArea, {
            lineNumbers: true,
            mode: 'shell',
        });
        curl_cm.setSize('100%', '100%');

        // set change handlers
        envoy_cm.on('change', cm => this.update(cm.getValue(), undefined))
        curl_cm.on('change', cm => this.update(undefined, cm.getValue()));
    },
    methods: {
        update: function(envoy_config, curl_command) {
            if (envoy_config) {
                this.envoy_config = envoy_config;
            }
            if (curl_command) {
                this.curl_command = curl_command;
            }
            localStorage.setItem('state', JSON.stringify({
                envoy_config: this.envoy_config,
                curl_command: this.curl_command,
            }));
        },

        copyURL: async function(event) {
            event.preventDefault();
            const state = {
                envoy_config: this.envoy_config,
                curl_command: this.curl_command,
            };
            const hash = btoa(JSON.stringify(state));
            window.location.hash = hash;
            // copy url to clipboard
            const url = window.location.href;
            const el = document.createElement('textarea');
            el.value = url;
            document.body.appendChild(el);
            el.select();
            document.execCommand('copy');
            document.body.removeChild(el);
            // quickly display "copied"
            this.copied = true;
            await sleep(1000);
            this.copied = false
        },

        loadDefault: async function() {
            const configs = await fetch('default_configs.json');
            this.default_configs = await configs.json();
            return {
                'envoy_config': this.default_configs['basic.conf'],
                'curl_command': 'http --pretty format get http://localhost:10000/get',
            }
        },

        loadFromLocalStorage: function() {
            try {
                return JSON.parse(localStorage.getItem('state'));
            } catch {
                return
            }
        },

        loadFromHash: function() {
            if (!window.location.hash) {
                return;
            }
            try {
                console.log(atob(decodeURIComponent(window.location.hash.substring(1))))
                return JSON.parse(atob(decodeURIComponent(window.location.hash.substring(1))));
            } catch {
                return;
            }
        },


        heights: function() {
            if (this.error && this.curl_result) {
                return ['h-1/3', 'h-1/3'];
            } else if (this.error) { // just error
                return ['hidden', 'h-2/3'];
            } else { // just curl
                return ['h-2/3', 'hidden'];
            }
        },

        error_height: function() {
            return this.heights()[1];
        },

        result_height: function() {
            return this.heights()[0];
        },

        run: async function(event) {
            event.preventDefault();
            this.loading = true;
            let response = await fetch('https://envoy-playground.fly.dev/api', {
                method: 'POST',
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    envoy_config: this.envoy_config,
                    command: this.curl_command,
                })
            });
            const json = await response.json();
            this.loading = false;
            this.error = json['error'];
            this.curl_result = json['result'];
        },
    }
})

app.mount("#app");

// on ready
