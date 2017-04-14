%{
package hariti
%}

%union{
    tok Token
    file *File
    bundles []Bundle
    bundle Bundle
    remoteBundle *RemoteBundle
    localBundle *LocalBundle
    bundleOptions BundleOptions
    bundleOption BundleOption
    aliases []string
    dependencies []string
    buildScripts []string
    buildScript string
}

%type<file> file
%type<bundles> bundles_zero_or_more
%type<bundle> bundle
%type<bundleOptions> remote_bundle_options_zero_or_more remote_bundle_options
%type<bundleOptions> local_bundle_options_zero_or_more local_bundle_options
%type<bundleOption> remote_bundle_option build_scripts_zero_or_more build_scripts
%type<aliases> aliases
%type<dependencies> dependencies_zero_or_more dependencies
%type<buildScripts> build_scripts_lines_zero_or_more build_scripts_lines
%type<buildScript> build_scripts_line
%token<tok> Use Local Ident As Depends EnableIf Build On OSType String

%%

file
: bundles_zero_or_more
  {
    $$= &File{
      Bundles: $1,
    }
    yylex.(*Lexer).Bundles= $1
  }
;

bundles_zero_or_more
: bundle bundles_zero_or_more
  {
    $$= append([]Bundle{$1}, $2...)
  }
|
  { $$= make([]Bundle, 0) }
;

bundle
: Use Ident remote_bundle_options_zero_or_more
  {
    $$= &RemoteBundle{
      Uri: $2.Text,
    }
    $3.Apply($$)
  }
| Use Local Ident local_bundle_options_zero_or_more
  {
    $$= &LocalBundle{
      Uri: $3.Text,
    }
    $4.Apply($$)
  }
;

remote_bundle_options_zero_or_more
: remote_bundle_options
  {
    $$= $1
  }
|
  {
    $$= make(BundleOptions, 0)
  }
;

remote_bundle_options
: remote_bundle_option remote_bundle_options
  {
    $$= append($2, $1)
  }
|
  {
    $$= make(BundleOptions, 0)
  }
;

remote_bundle_option
: As aliases
  {
    $$= &AliasesOption{
      Value: $2,
    }
  }
| EnableIf String
  {
    $$= &EnableIfExprOption{
      Value: $2.Text,
    }
  }
| Depends '(' dependencies_zero_or_more ')'
  {
    $$= &DependenciesOption{
      Value: $3,
    }
  }
| Build '{' build_scripts_zero_or_more '}'
  {
    opts := make([]BundleOption, 0)
    opts = append(opts, $3)
    $$ = BundleOptions(opts)
  }
;

aliases
: Ident ',' aliases
  {
    $$= append($3, $1.Text)
  }
| Ident
  {
    $$= []string{$1.Text}
  }
;

dependencies_zero_or_more
: dependencies
  {
    $$= $1
  }
|
  {
    $$= []string{}
  }
;

dependencies
: Ident dependencies
  {
    $$= append([]string{$1.Text}, $2...)
  }
|
  {
    $$= []string{}
  }
;

build_scripts_zero_or_more
: build_scripts
  {
    $$= $1
  }
|
  {
    $$= &BuildScriptsOption{
      Value: make(map[string][]string, 0),
    }
  }
;

build_scripts
: On OSType build_scripts_lines_zero_or_more build_scripts
  {
    $$= &BuildScriptsOption{
      Value: map[string][]string{
        $2.Text: $3,
      },
    }
    for ostype, scripts := range $4.(*BuildScriptsOption).Value {
      $$.(*BuildScriptsOption).Value[ostype] = scripts
    }
  }
|
  {
    $$= &BuildScriptsOption{
      Value: make(map[string][]string, 0),
    }
  }
;

build_scripts_lines_zero_or_more
: build_scripts_lines
  {
    $$= $1
  }
|
  {
    $$= []string{}
  }
;

build_scripts_lines
: build_scripts_line build_scripts_lines
  {
    $$= append([]string{$1}, $2...)
  }
|
  {
    $$= []string{}
  }
;

build_scripts_line
: '-' Ident '\n'
  {
    $$= $2.Text
  }
;

local_bundle_options_zero_or_more
: local_bundle_options
  {
    $$= $1
  }
|
  {
    $$= make(BundleOptions, 0)
  }
;

local_bundle_options
:
  {
    $$= make(BundleOptions, 0)
  }
;

%%
