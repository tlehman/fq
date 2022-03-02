include "internal";
include "query";


def _eval_error($what; $error):
  error({
    what: $what,
    error: $error,
    column: 0,
    line: 1,
    filename: ""
  });

def _eval_error_function_not_defined($name; $args):
  _eval_error(
    "compile";
    "function not defined: \($name)/\($args | length)"
  );

def _eval_query_rewrite($opts):
  _query_fromtostring(
    ( . as $orig_query
    | _query_pipe_last as $last
    | ( $last
      | if _query_is_func then [_query_func_name, _query_func_args]
        else ["", []]
        end
      ) as [$last_func_name, $last_func_args]
    | $opts.slurps[$last_func_name] as $slurp
    | if $slurp then
        _query_transform_pipe_last(_query_ident)
      end
    | if $opts.catch_query then
        # _query_query to get correct precedence and a valid query
        # try (1+1) catch vs try 1 + 1 catch
        _query_try(. | _query_query; $opts.catch_query)
      end
    | _query_pipe(
        $opts.input_query // _query_ident;
        .
      )
    | if $slurp then
        _query_func(
          $slurp;
          [ # pass original, rewritten and args queries as query ast trees
            ( { slurp: _query_string($last_func_name)
              , slurp_args:
                  ( $last_func_args
                  | if . then
                      ( map(_query_toquery)
                      | _query_commas
                      | _query_array
                      )
                    else (null | _query_array)
                    end
                  )
              , orig: ($orig_query | _query_toquery)
              , rewrite: _query_toquery
              }
            | _query_object
            )
          ]
        )
      end
    )
  );

# TODO: better way? what about nested eval errors?
def _eval_is_compile_error:
  type == "object" and .error != null and .what != null;
def _eval_compile_error_tostring:
  [ (.filename // "expr")
  , if .line != 1 or .column != 0 then "\(.line):\(.column)"
    else empty
    end
  , " \(.error)"
  ] | join(":");

def eval($expr; $opts; on_error; on_compile_error):
  ( . as $c
  | ($opts.filename // "expr") as $filename
  | try
      _eval(
        $expr | _eval_query_rewrite($opts);
        $filename
      )
    catch
      if _eval_is_compile_error then
        # rewrite parse error will not have filename
        ( .filename = $filename
        | {error: ., input: $c}
        | on_compile_error
        )
      else
        ( {error: ., input: $c}
        | on_error
        )
      end
  );
def eval($expr): eval($expr; {}; .error | error; .error | error);
