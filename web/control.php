<?php

		function _str_send($sock, $str) {
			return socket_send($sock, $str, strlen($str), 0);
		}

		function get_json($tp) {
			$sock = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
			if (!socket_connect($sock, '127.0.0.1', '9110')) {
				throw new Exception("couldn't connect");
			}

			_str_send($sock, $tp);

			$buf = "";
			socket_recv($sock, $buf, 1024, 0);

			$x = json_decode($buf, true);

			socket_close($sock);

			return $x;
		}

		function get_settings() {
			return get_json("SETTINGS\n");
		}

		function get_status() {
			return get_json("STATUS\n");
		}

		function set_settings($to_set) {
			$sock = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
			if (!socket_connect($sock, '127.0.0.1', '9110')) {
				print "couldn't connect\n";
				exit;
			}

			$buf = json_encode($to_set);
			_str_send($sock, $buf . "\n");
			_str_send($sock, "QUIT\n");
		}

		$current_settings = get_settings();

		// Copy over old settings to the new ones, such that if there is a field that doesn't change it gets sent over again.
		if (isset($_SERVER['REQUEST_METHOD']) && ($_SERVER['REQUEST_METHOD'] == 'POST')) {
			$new_settings = $current_settings;
			foreach ($_POST as $k => $v) {
				if ($v === "true") $v = true;
				else if ($v === "false") $v = false;
				$new_settings[$k] = $v; //FIXME.
			}
			set_settings($new_settings);
			$current_settings = get_settings();
		}

		$current_status = get_status();

		$p = array_merge($current_settings, $current_status);

		print json_encode($p) . "\n";
?>
