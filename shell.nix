{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
	buildInputs = with pkgs; [
		go
		gopls
		go-tools
	];

	CGO_ENABLED = "1";
}
